package handler

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/config/db"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/logger"
	decryptService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/decryptService/service"
	dumpMetricService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/dumpMetricService/v0"
	getFlatService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/getFlatService/v0"
	getService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/getService/v0"
	hashService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/hashService/v0"
	listMetricService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/listMetricService/v0"
	subnetService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/subnetService/service"
	updateBatchService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/updateBatchService/service"
	updateFlatService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/updateFlatService/v0"
	updateService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/updateService/v0"
)

type Route struct {
	Method  string
	Path    string
	Handler http.Handler
}

type API struct {
	router *chi.Mux
	logger logger.HTTPLogger

	updateFlatService  *updateFlatService.Service
	updateBatchService updateBatchService.UpdateBatchService
	updateService      *updateService.Service

	getFlatService *getFlatService.Service
	getService     *getService.Service

	listMetricService *listMetricService.Service

	dumpSyncMetricService *dumpMetricService.Service
	hashService           *hashService.Service

	decryptService decryptService.DecryptService
	subnetService  subnetService.SubnetService
}

func New(
	logger logger.HTTPLogger,
	updateFlatService *updateFlatService.Service,
	updateBatchService updateBatchService.UpdateBatchService,
	updateService *updateService.Service,
	getFlatService *getFlatService.Service,
	getService *getService.Service,
	listMetricService *listMetricService.Service,
	dumpSyncMetricService *dumpMetricService.Service,
	hashService *hashService.Service,
	decryptService decryptService.DecryptService,
	subnetService subnetService.SubnetService,
) *API {
	return &API{
		router:                chi.NewRouter(),
		logger:                logger,
		updateFlatService:     updateFlatService,
		updateBatchService:    updateBatchService,
		updateService:         updateService,
		getFlatService:        getFlatService,
		getService:            getService,
		listMetricService:     listMetricService,
		dumpSyncMetricService: dumpSyncMetricService,
		hashService:           hashService,
		decryptService:        decryptService,
		subnetService:         subnetService,
	}
}

func (api API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.router.ServeHTTP(w, r)
}

func (api API) RegisterPing(db *db.PGConnect) {
	api.router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			WriteError(w, err)
		}

		WriteOK(w)
	})
}

func (api API) RegisterHandlers() {
	api.router.Group(func(r chi.Router) {
		r.Get("/", DoListMetricResponse(api.listMetricService.Do).ServeHTTP)
	})

	api.router.Group(func(r chi.Router) {
		r.Use(api.WithLogging)
		r.Use(MiddlewareMetricName)
		r.Post("/update/{type}/{name}/{value}", func(w http.ResponseWriter, rq *http.Request) {
			DoUpdateFlatResponse(
				api.updateFlatService.Do, chi.URLParam(rq, "type"), chi.URLParam(rq, "name"), chi.URLParam(rq, "value"),
			).ServeHTTP(w, rq)
		})
	})

	api.router.Group(func(r chi.Router) {
		r.Use(api.WithLogging)
		r.Use(api.WithSync)
		r.Post("/update/", DoUpdateJSONResponse(api.updateService.Do).ServeHTTP)
	})

	api.router.Group(func(r chi.Router) {
		r.Use(api.WithLogging)
		r.Use(api.WithSync)
		r.Post("/updates/", DoUpdateBatchJSONResponse(api.updateBatchService.Do).ServeHTTP)
	})

	api.router.Group(func(r chi.Router) {
		r.Use(api.WithLogging)
		r.Use(MiddlewareMetricName)
		r.Get("/value/{type}/{name}", func(w http.ResponseWriter, rq *http.Request) {
			DoGetFlatResponse(
				api.getFlatService.Do, chi.URLParam(rq, "type"), chi.URLParam(rq, "name"),
			).ServeHTTP(w, rq)
		})
	})

	api.router.Group(func(r chi.Router) {
		r.Use(api.WithLogging)
		r.Post("/value/", DoGetJSONResponse(api.getService.Do).ServeHTTP)
	})
}

func (api API) WithLogging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var resp ResponseInfo
		rw := responseWriter{
			ResponseWriter: w,
			response:       &resp,
		}

		h.ServeHTTP(&rw, r)

		api.logger.LogHTTP(logger.HTTPInfo{
			URI:      r.RequestURI,
			Method:   r.Method,
			Duration: time.Since(start),
			Response: logger.ResponseInfo{
				Size:   resp.Size,
				Status: resp.Status,
				Body:   resp.Body,
			},
		})
	})
}

func (api API) RegisterPprof() {
	api.router.Group(func(r chi.Router) {
		r.Use(MiddlewareLocalhost)
		r.Mount("/debug", middleware.Profiler())
	})
}

func (api API) WithSync(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)

		if err := api.dumpSyncMetricService.WriteDump(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func (api API) WithHash(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hw := w
		if hash := r.Header.Get("HashSHA256"); hash != "" {
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, r.Body); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			if err := api.hashService.Validate(r.Context(), buf.Bytes(), hash); err != nil {
				WriteError(w, err)
				return
			}

			r.Body = io.NopCloser(buf)

			hw = &responseHashWriter{
				ResponseWriter: w,
				body:           bytes.Buffer{},
				hashFunc: func(message []byte) (string, error) {
					return api.hashService.Hash(r.Context(), message)
				},
			}
		}

		h.ServeHTTP(hw, r)
	})
}

func (api API) WithDecrypt(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encrypted, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		decrypted, err := api.decryptService.Decrypt(r.Context(), encrypted)
		if err != nil {
			WriteError(w, err)
			return
		}

		r.Body = io.NopCloser(bytes.NewBuffer(decrypted))

		h.ServeHTTP(w, r)
	})
}

func (api API) WithTrustedSubnet(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			if err := api.subnetService.Validate(r.Context(), net.ParseIP(realIP)); err != nil {
				WriteError(w, err)
				return
			}
		}

		h.ServeHTTP(w, r)
	})
}

func (api API) AddRoute(route Route) {
	if route.Method == "" {
		api.router.Handle(route.Path, route.Handler)
	} else {
		api.router.Handle(route.Method+" "+route.Path, route.Handler)
	}
}

func (api API) AddRoutes(routes []Route) {
	for _, r := range routes {
		api.AddRoute(r)
	}
}
