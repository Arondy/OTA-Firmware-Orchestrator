package rollout_controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	healthv1 "github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/health/v1"
	"github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/health/v1/healthv1connect"
	"github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/rollout/v1/rolloutv1connect"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/clients/rollout-controller/interceptors"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"go.uber.org/zap"
)

type Client struct {
	campaignClient rolloutv1connect.CampaignServiceClient
	httpClient     *http.Client
}

func NewClient(config config.RolloutControllerConfig, logger *zap.SugaredLogger) (*Client, error) {
	logger.Debugf("Connecting to Rollout Controller on %s:%d", config.Host, config.Port)

	httpClient := &http.Client{Timeout: config.Timeout}
	baseURL := fmt.Sprintf("%s://%s:%d", config.Scheme, config.Host, config.Port)
	if err := CheckHealth(httpClient, baseURL); err != nil {
		return nil, err
	}

	interceptors := interceptors.NewInterceptorsOption()

	campaignClient := rolloutv1connect.NewCampaignServiceClient(httpClient, baseURL, interceptors)

	return &Client{
		campaignClient: campaignClient,
		httpClient:     httpClient,
	}, nil
}

func CheckHealth(httpClient *http.Client, baseURL string) error {
	const attempts = 15
	const pause = time.Second
	var lastErr error

	health := healthv1connect.NewHealthServiceClient(httpClient, baseURL)

	for i := range attempts {
		_, err := health.CheckHealth(context.Background(), connect.NewRequest(&healthv1.CheckHealthRequest{}))
		if err == nil {
			return nil
		}

		lastErr = fmt.Errorf("connection to Rollout Controller failed after %d attempts: %w", attempts, err)
		if i < attempts-1 {
			time.Sleep(pause)
		}
	}
	return lastErr
}

func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}
