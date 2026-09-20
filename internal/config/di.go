package config

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/config/db"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/grpc/api"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/grpc/server"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/handler"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/logger"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/repository/audit/file"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/repository/audit/remote"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/repository/encode"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/repository/outbox"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/repository/storage/inmemory"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/repository/storage/pg"
	auditFileService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/auditFileService/v0"
	auditRemoteService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/auditRemoteService/v0"
	decryptService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/decryptService/service"
	decryptServiceV0 "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/decryptService/v0"
	dumpMetricService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/dumpMetricService/v0"
	getCounterService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/getCounterService/v0"
	getFlatService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/getFlatService/v0"
	getGaugeService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/getGaugeService/v0"
	getService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/getService/v0"
	hashService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/hashService/v0"
	listMetricService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/listMetricService/v0"
	subnetService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/subnetService/service"
	subnetServiceV0 "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/subnetService/v0"
	updateBatchServiceRemote "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/updateBatchService/remote"
	updateBatchService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/updateBatchService/service"
	updateBatchServiceV0 "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/updateBatchService/v0"
	updateCounterService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/updateCounterService/v0"
	updateFlatService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/updateFlatService/v0"
	updateGaugeService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/updateGaugeService/v0"
	updateService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/updateService/v0"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/worker/sworker"
	"github.com/MaksimMakarenko1001/ya-go-advanced/pkg/backoff"
)

type DI struct {
	config       *diConfig
	logger       *logger.ZapLogger
	httpServer   *http.Server
	grpcServer   *server.Server
	repositories struct {
		encoder         *encode.JSONEncode
		inmemoryStorage *inmemory.Repository
		pgStorage       *pg.Repository
		outbox          *outbox.Repository
		fileAuditor     *file.Repository
		remoteAuditor   *remote.Repository
	}
	services struct {
		included struct {
			updateCounterService *updateCounterService.Service
			updateGaugeService   *updateGaugeService.Service

			getCounterService *getCounterService.Service
			getGaugeService   *getGaugeService.Service
		}
		internal struct {
			updateBatchRemoteService updateBatchService.UpdateBatchRemoteService
		}
		updateFlatService  *updateFlatService.Service
		updateBatchService updateBatchService.UpdateBatchService
		updateService      *updateService.Service

		getFlatService *getFlatService.Service
		getService     *getService.Service

		listMetricService *listMetricService.Service

		dumpMetricService     *dumpMetricService.Service
		dumpSyncMetricService *dumpMetricService.Service

		hashService    *hashService.Service
		decryptService decryptService.DecryptService
		subnetService  subnetService.SubnetService

		auditFileService   *auditFileService.Service
		auditRemoteService *auditRemoteService.Service
	}
	workers struct {
		auditFile   *sworker.SimpleWorker
		auditRemote *sworker.SimpleWorker
	}
	api struct {
		external *handler.API
		internal *api.API
	}
	infr struct {
		db *db.PGConnect
	}
}

func (di *DI) Init(envPrefix string) {
	di.config = &diConfig{}
	di.config.loadConfig(envPrefix)

	di.InitLogging()
	di.initDB()
	di.initRepositories()
	di.initServices()
	di.initWorkers()
	di.initAPI()
	di.initGRPC()
}

func (di *DI) InitLogging() {
	var err error
	di.logger, err = logger.New(di.config.Logger)
	if err != nil {
		log.Println("logger init not ok,", err.Error())
	}
}

func (di *DI) initDB() {
	var err error
	di.infr.db, err = db.New(
		di.config.Database,
		backoff.NewBackoff(di.config.Database.MaxRetries, db.ClassifyPgError),
	)
	if err != nil {
		log.Println("db init not ok,", err.Error())
	}
}

func (di *DI) initRepositories() {
	di.repositories.encoder = encode.New()
	di.repositories.inmemoryStorage = inmemory.New(di.repositories.encoder)
	di.repositories.pgStorage = pg.New(di.infr.db, di.repositories.inmemoryStorage)
	di.repositories.outbox = outbox.New(di.infr.db)
	di.repositories.fileAuditor = file.New(di.config.AuditFile)
	di.repositories.remoteAuditor = remote.New(di.config.AuditRemote)
}

func (di *DI) initServices() {
	di.services.included.updateCounterService = updateCounterService.New(di.repositories.pgStorage)
	di.services.included.updateGaugeService = updateGaugeService.New(di.repositories.pgStorage)

	di.services.included.getCounterService = getCounterService.New(di.repositories.pgStorage)
	di.services.included.getGaugeService = getGaugeService.New(di.repositories.pgStorage)

	di.services.updateFlatService = updateFlatService.New(di.services.included.updateCounterService,
		di.services.included.updateGaugeService)
	di.services.updateBatchService = updateBatchServiceV0.New(di.repositories.pgStorage)
	di.services.updateService = updateService.New(di.services.included.updateCounterService,
		di.services.included.updateGaugeService)

	di.services.getFlatService = getFlatService.New(di.services.included.getCounterService,
		di.services.included.getGaugeService)
	di.services.getService = getService.New(di.services.included.getCounterService,
		di.services.included.getGaugeService)

	di.services.listMetricService = listMetricService.New(di.repositories.pgStorage)

	di.services.dumpMetricService = dumpMetricService.New(di.config.DumpService, di.config.FileStoragePath, di.repositories.inmemoryStorage)
	di.services.dumpSyncMetricService = dumpMetricService.New(di.config.DumpSyncService, di.config.FileStoragePath, di.repositories.inmemoryStorage)

	di.services.hashService = hashService.New(di.config.HashService)

	di.services.decryptService = decryptServiceV0.New(di.config.DecryptService)
	di.services.subnetService = subnetServiceV0.New(di.config.SubnetService)

	di.services.auditFileService = auditFileService.New(di.config.AuditFileService, di.repositories.outbox, di.repositories.fileAuditor)
	di.services.auditRemoteService = auditRemoteService.New(di.config.AuditRemoteService, di.repositories.outbox, di.repositories.remoteAuditor)

	di.services.internal.updateBatchRemoteService = updateBatchServiceRemote.New(di.services.updateBatchService)
}

func (di *DI) initWorkers() {
	di.workers.auditFile = sworker.New(
		di.config.Worker.AuditFile,
		"audit_file",
		di.services.auditFileService.Do,
	)
	di.workers.auditRemote = sworker.New(
		di.config.Worker.AuditRemote,
		"audit_remote",
		di.services.auditRemoteService.Do,
	)
}

func (di *DI) initAPI() {
	di.api.external = handler.New(
		di.logger,
		di.services.updateFlatService,
		di.services.updateBatchService,
		di.services.updateService,
		di.services.getFlatService,
		di.services.getService,
		di.services.listMetricService,
		di.services.dumpSyncMetricService,
		di.services.hashService,
		di.services.decryptService,
		di.services.subnetService,
	)

	di.api.internal = api.New(
		di.services.internal.updateBatchRemoteService,
		di.services.subnetService,
	)
}

func (di *DI) initGRPC() {
	di.grpcServer = server.New(
		di.config.GRPC,
		di.api.internal,
		di.api.internal.WithTrustedSubnet(),
	)
}

func (di *DI) Start(errorCh chan<- error, certFile string, keyFile string) {
	ctx, cancel := context.WithCancel(context.Background())

	di.workers.auditFile.Start(ctx)
	di.workers.auditRemote.Start(ctx)

	if di.config.Restore {
		if err := di.services.dumpMetricService.ReadDump(); err != nil {
			log.Println(err.Error())
		}
	}

	if di.config.StoreInterval > 0 {
		go di.doDump(ctx)
	}

	di.grpcServer.Start(errorCh)

	di.api.external.RegisterPing(di.infr.db)
	di.api.external.RegisterHandlers()
	di.api.external.RegisterPprof()

	di.httpServer = &http.Server{
		Addr: di.config.HTTP.Address,
		Handler: handler.Conveyor(
			di.api.external,
			handler.MiddlewareCompress,
			di.api.external.WithHash,
			di.api.external.WithDecrypt,
			di.api.external.WithTrustedSubnet,
		),
	}

	go func() {
		defer cancel()

		if err := di.httpServer.ListenAndServeTLS(certFile, keyFile); !errors.Is(err, http.ErrServerClosed) {
			errorCh <- err
		}
	}()
}

func (di *DI) Stop(ctx context.Context) {
	di.httpServer.Shutdown(ctx)
	di.grpcServer.Shutdown(ctx)
	di.infr.db.Close()
	di.repositories.fileAuditor.FileClose(context.TODO())
}

func (di *DI) doDump(ctx context.Context) {
	ticker := time.NewTicker(di.config.StoreInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := di.services.dumpMetricService.WriteDump(); err != nil {
				log.Println(err.Error())
			}
		}
	}
}
