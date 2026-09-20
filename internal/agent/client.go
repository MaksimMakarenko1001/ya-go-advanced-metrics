package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	pb "github.com/MaksimMakarenko1001/ya-go-advanced/api/proto/metrics"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/grpc/client"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/models"
	"github.com/MaksimMakarenko1001/ya-go-advanced/pkg"
	"github.com/MaksimMakarenko1001/ya-go-advanced/pkg/backoff"
)

type Client struct {
	config     Config
	httpClient *http.Client
	grpcClient *client.Client
	memStats   runtime.MemStats
	pollCount  int64
	backoff    *backoff.Backoff
	cryptoKey  *rsa.PublicKey
}

func NewClient(cfg Config) *Client {
	httpClient := &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return &Client{
		config:     cfg,
		httpClient: httpClient,
		grpcClient: client.New(cfg.GRPC).WithRealIP(cfg.RealIP),
		backoff: backoff.NewBackoff(
			cfg.MaxRetries,
			ClassifyHTTPError,
		),
	}
}

func (c *Client) sendBatchJSON(batch []models.Metric) (err error) {
	if len(batch) == 0 {
		return nil
	}

	u := url.URL{
		Scheme: "https",
		Host:   c.config.Address,
		Path:   "/updates/",
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(batch)
	if err != nil {
		return fmt.Errorf("batch encoder not ok, %w", err)
	}

	r, err := newGZipRequest(http.MethodPost, u.String(), buf.Bytes())
	if err != nil {
		return fmt.Errorf("batch request not ok, %w", err)
	}

	var status int
	fn := func(ctx context.Context) error {
		var sendErr error
		status, sendErr = c.send(r)
		return sendErr
	}

	backoff := c.backoff.WithLinear(time.Second, time.Second*2)
	err = backoff(fn)(context.Background())
	if err != nil {
		return fmt.Errorf("batch http not ok, %w", err)
	}

	if status != http.StatusOK {
		return fmt.Errorf("batch response status not ok, %s: %d", buf.String(), status)
	}

	return nil
}

func (c *Client) sendBatchGRPC(batch []models.Metric) error {
	if len(batch) == 0 {
		return nil
	}

	req := pb.UpdateMetricsRequest_builder{
		Metrics: make([]*pb.Metric, 0, len(batch)),
	}

	for _, m := range batch {
		metric := pb.Metric_builder{Id: m.ID}

		switch m.MType {
		case pkg.MetricTypeCounter:
			metric.Type = pb.Metric_COUNTER
			metric.Delta = *m.Delta
		case pkg.MetricTypeGauge:
			metric.Type = pb.Metric_GAUGE
			metric.Value = *m.Value
		}

		req.Metrics = append(req.Metrics, metric.Build())
	}

	_, err := c.grpcClient.UpdateMetrics(context.Background(), req.Build())

	return err
}

func (c *Client) sendGaugeMetric(metricName string, value float64) (err error) {
	valueStr := strconv.FormatFloat(value, 'f', -1, 64)

	u := url.URL{
		Scheme: "http",
		Host:   c.config.Address,
		Path:   "/update/gauge/" + metricName + "/" + valueStr,
	}

	resp, err := http.Post(u.String(), "text/plain", nil)
	if err != nil {
		return fmt.Errorf("error sending gauge metric: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send gauge metric %s: %d", metricName, resp.StatusCode)
	}

	return nil
}

func (c *Client) sendGaugeMetricJSON(metricName string, value float64) (err error) {
	u := url.URL{
		Scheme: "http",
		Host:   c.config.Address,
		Path:   "/update/",
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(models.Metric{
		ID:    metricName,
		MType: pkg.MetricTypeGauge,
		Value: &value,
	})
	if err != nil {
		return fmt.Errorf("gauge encoder not ok, %w", err)
	}

	r, err := newGZipRequest(http.MethodPost, u.String(), buf.Bytes())
	if err != nil {
		return fmt.Errorf("gauge request not ok, %w", err)
	}

	status, err := c.send(r)
	if err != nil {
		return fmt.Errorf("gauge http not ok, %w", err)
	}

	if status != http.StatusOK {
		return fmt.Errorf("gauge response status not ok, %s: %d", metricName, status)
	}

	return nil
}

func (c *Client) sendCounterMetric(metricName string, value int64) (err error) {
	valueStr := strconv.FormatInt(value, 10)

	u := url.URL{
		Scheme: "http",
		Host:   c.config.Address,
		Path:   "/update/counter/" + metricName + "/" + valueStr,
	}

	resp, err := http.Post(u.String(), "text/plain", nil)
	if err != nil {
		return fmt.Errorf("error sending counter metric: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send counter metric %s: %d", metricName, resp.StatusCode)
	}

	return nil
}

func (c *Client) sendCounterMetricJSON(metricName string, value int64) (err error) {

	u := url.URL{
		Scheme: "http",
		Host:   c.config.Address,
		Path:   "/update/",
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(models.Metric{
		ID:    metricName,
		MType: pkg.MetricTypeCounter,
		Delta: &value,
	})

	if err != nil {
		return fmt.Errorf("counter encoder not ok, %w", err)
	}

	r, err := newGZipRequest(http.MethodPost, u.String(), buf.Bytes())
	if err != nil {
		return fmt.Errorf("counter request not ok, %w", err)
	}

	status, err := c.send(r)
	if err != nil {
		return fmt.Errorf("counter http not ok, %w", err)
	}

	if status != http.StatusOK {
		return fmt.Errorf("counter response status not ok, %s: %d", metricName, status)
	}

	return nil
}

func (c *Client) collect(doneCh <-chan struct{}) <-chan models.Metric {
	ch := make(chan models.Metric)
	genChs := []<-chan models.Metric{}

	collection := []models.Metric{}
	poolTicker := time.NewTicker(c.config.PollInterval)
	reportTicker := time.NewTicker(c.config.ReportInterval)

	go func() {
		defer close(ch)
		defer poolTicker.Stop()
		defer reportTicker.Stop()

		for {
			select {
			case <-doneCh:
				log.Println("Stop collecting")
				return
			case <-poolTicker.C:
				log.Println("Collects metrics")

				collection = append(collection, genCounters(&c.pollCount)...)
				collection = append(collection, genGauge(&c.memStats)...)
				collection = append(collection, genExtraGauge()...)
				genChs = append(genChs, gen(doneCh, collection))
			case <-reportTicker.C:
				log.Println("Try to report metrics")

				inCh := fanIn(doneCh, genChs)
				genChs = genChs[:0]

				go func(channel <-chan models.Metric) {
					for i := range channel {
						select {
						case <-doneCh:
							return
						case ch <- i:
						}
					}
				}(inCh)
			}
		}
	}()

	return ch
}

func (c *Client) sendWorker(id int, batchedCh <-chan []models.Metric, results chan<- string) {
	for batch := range batchedCh {
		res := fmt.Sprintf("#%d: success", id)

		err := c.sendBatchJSON(batch)
		if err != nil {
			res = fmt.Sprintf("#%d: fail, %s", id, err.Error())
		}

		results <- res
	}
	results <- fmt.Sprintf("#%d: done", id)
}

func (c *Client) sendWorkerGRPC(id int, batchedCh <-chan []models.Metric, results chan<- string) {
	for batch := range batchedCh {
		res := fmt.Sprintf("grpc#%d: success", id)

		err := c.sendBatchGRPC(batch)
		if err != nil {
			res = fmt.Sprintf("grpc#%d: fail, %s", id, err.Error())
		}

		results <- res
	}
	results <- fmt.Sprintf("grpc#%d: done", id)
}

func (c *Client) Run(ctx context.Context) error {
	doneCh := make(chan struct{})

	metricCh := c.collect(doneCh)
	batchedCh := batched(metricCh, c.config.BatchSize)
	results := make(chan string, c.config.RateLimit)

	var wg sync.WaitGroup
	// http and grpc work simultaneously
	for w := range c.config.RateLimit {
		wg.Go(func() {
			c.sendWorker(w, batchedCh, results)
		})

		wg.Go(func() {
			c.sendWorkerGRPC(w, batchedCh, results)
		})
	}

	go func() {
		for res := range results {
			log.Println(res)
		}
		log.Println("done")
	}()

	<-ctx.Done()
	// stop collecting
	close(doneCh)

	wg.Wait()
	// stop iterating results
	close(results)
	return ctx.Err()

}

func (c *Client) send(req *http.Request) (int, error) {
	buf := bytes.NewBuffer(nil)
	_, err := io.Copy(buf, req.Body)
	if err != nil {
		return 0, fmt.Errorf("copy not ok, %w", err)
	}

	hash, err := c.hashUp(buf.Bytes())
	if err != nil {
		return 0, fmt.Errorf("hash not ok, %w", err)
	}

	encrypted, err := c.encryptUp(buf.Bytes())
	if err != nil {
		return 0, err
	}

	r, err := http.NewRequest(req.Method, req.URL.String(), bytes.NewBuffer(encrypted))
	if err != nil {
		return 0, fmt.Errorf("request not ok, %w", err)
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Content-Encoding", "gzip")
	r.Header.Set("Accept-Encoding", "gzip")
	r.Header.Set("HashSHA256", hash)
	r.Header.Set("X-Real-IP", c.config.RealIP)

	resp, err := c.httpClient.Do(r)

	if err != nil {
		return 0, fmt.Errorf("http not ok, %w", err)
	}

	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func newGZipRequest(method string, url string, body []byte) (*http.Request, error) {
	buf := bytes.NewBuffer(nil)
	zw := gzip.NewWriter(buf)

	_, err := zw.Write(body)
	if err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	request, err := http.NewRequest(method, url, buf)
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (c *Client) hashUp(body []byte) (string, error) {
	h := hmac.New(sha256.New, []byte(c.config.Key))
	if _, err := h.Write(body); err != nil {
		return "", fmt.Errorf("failed to hash message, %w", err)
	}

	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func (c *Client) encryptUp(body []byte) ([]byte, error) {
	if c.cryptoKey == nil {
		return body, nil
	}

	encrypted, err := rsa.EncryptOAEP(md5.New(), rand.Reader, c.cryptoKey, body, nil)
	if err != nil {
		return nil, fmt.Errorf("encrypt message error: %w", err)
	}

	return encrypted, nil
}

func (c *Client) WithCrypto(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read crypto key error: %w", err)
	}

	pemBlock, _ := pem.Decode(data)
	if pemBlock == nil {
		return errors.New("crypto key not found")
	}

	private, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		return fmt.Errorf("parse crypto key error: %w", err)
	}

	c.cryptoKey = &private.PublicKey
	return nil
}
