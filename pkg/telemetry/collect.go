package telemetry

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/telemetry/serviceaccount"
	"github.com/loft-sh/devpod/pkg/telemetry/types"
	"github.com/spf13/cobra"
	"gopkg.in/square/go-jose.v2/jwt"
)

const (
	telemetryEndpoint          = "https://admin.loft.sh/analytics/v1/devpod/v1/events"
	telemetryRequestTimeout    = 5 * time.Second
	telemetrySendFinishedAfter = 10 * time.Second
)

var (
	Collector EventCollector = NewDefaultCollector()

	UIEventsExceptions []string = []string{"devpod list", "devpod provider list", "devpod status"}

	// a dummy key so this doesn't fail for dev/testing, this is set by build flag in release action
	telemetryPrivateKey = `LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlKS1FJQkFBS0NBZ0VBdlE3cHhqYzEybzlJdXQyQkQ2TUtaWnhDY29hbXpJNVV0Wll6Wk5GZFVQYkJsSlI0ClBXOEM1STM3ZHk1cW1yMlU5UlJSbjNlOUpjSDRPS0QzenVHSkhDd0Z2TnpOYzJsYVQ0dlE5NjlVeVpmakdhT3AKVmxtSEhDaDJXajZvbHNUNmhldGJySTNpYzNvVm1XRHBhSHM4OGU3K2dzTnkyTUowNjNES0ZYM0VLV3pNQVVWZQprZUI1M29DWStCT0R0RExRcHd3eC9wQWp1bUFNS0dkNEc2a3FhcE1VZElpN1NKMzlyL2JxL2VUeWZwSUUzOW9ZCmoxanlhdkFpRFMxR1g0Mm5mU0lkck5NSDhERytSSzNVSHMycTFDOXI0Y1dzenVURktlOHprZ2ltdC9oY2sxS2sKZjZBTllyRE0vQmlrcWZoYXVDcHlMdDFhOTdHbUNNb0x1Z3NBWlB6TjhlTXk4eFlwZy9PZ3hjQ3d2a1E2SzVnTQo0Q3k5ZG5aOFVJTlBTVE02Z2xJUFR1cHpPUzYwVDlBb2VZREcwa245bEVYKzhHT1RyZ1NmaTBONEpxbzZ0YlU0Cjk0TXlvcTB2VmtaVjNGYWNKMDcwbFJQaUFxcm1BeDdMS1J0UUNoeiszY0hLcDhBUmNWS1RuT3VpSHdDNEk0SUYKUGhmbzZ4QWFQWUJ0RDRNQUtydFErS3pycVlwNHZaTVZZWUpWR1hzT2xFN1FGSUJjUk5FUTlPdmlTSHZpTEhsTApHbjhER1NxWVVBWEVPVVZBNWtkTUVwOGhYMXJkb2pyNk9FbEt5dnF1Wk9JMk9yekhDWmFCNS9nQVZwelRGcGhYCjlrN2oyU1FyNUcxbXI2TTE3VXJacnhBcUphSGo1ZnBuL2dCOW9ubkN1bmcvSTNDOEVzb0xVQnovd0xrQ0F3RUEKQVFLQ0FnRUFsMGVNcHBCZEpuTks5a1B5VnVuV2t2SVRkWUxyaTNsRXJUendDUGRDM1Z0bUVSY3drNi8xdDU4cApIZmZsVThicG42WlBuZlA1UlhKTnhqcC9zR3BtQlVYd25XeHRkYkZTazU1RWF6MC84a1A0Yy9heXRLYlU1eUkxCmVnYnpiaGxXZ2J5UDBhYURFbllaUEc4QXRoc083R1NhQVZhVjJuN1hnZUh4d25xdGNaeGVMWkl0bHpyeEthcnIKUEc2WkQ2TXR0TTJjWDU5RkI0aDlrZ01oWjdqWWVRa1I4Q0hOQXRGeFF0R292ZHJxYzM4eUtWRmlINnBENkhBWQpQMFVBTDh1d3Z2K0NrVjBYMkFwbHZwejl4Rnc4R3FlTGd0QmpjL1k1RWxJV2lQOGxNTWFxaFRRMjd1ekthVE1pCkE0TlFsN1ZrR2tQVXRFMXAwaE96MFFxamtZM21GSWpsTEhJYTBWN3QybmxrSXJNKzVLbVNqbFF4WjhMRHZGcnoKTkxFaUk5dnp1RWNHdVpPYXZHdjhzSmhXK2paQ3JQWVVyc3dPMTBsYnVsMkdnL2JBVFd2U3lVUGlyb1RsbEc5NApsVzFrMzl1MUk5d0tLaExSaE5TMlZQTW4xSE1CY0FFQXlTTWFudDdwbjQ4R1R1Q0VseUlEZTM4OW9KWHVOcXpNCjdrS2VaaG0wYzBJNmpvSThVcmNyMVZvTTErbmdtdzlxWldtOWJXekZpNW1IaTlCRXkyRGpjeHlOK1l3bFRVQW4Kd0EyblpoMVY0U1hUZUVWUzFOQ3J5dGNXSlZBdjJObWpTV3ZUQk4yaFdCZzEyVGpXZi9MSEF4eklyVzBaa0tKcgptVXdDQ0V3anhkM2JIMzluWnZkeG5xTXVISnZnUUpmM1NGQVRmZlZDWjI2OWdCZUtoZ0VDZ2dFQkFOc2IyTnZ5CmhxU2MwbW43VWtncHYvUjh0TVNsVVRId3g3cDNCZ21xR2tnU0JCR0t3cEI4Q1NzNWl4UHFGYzZaMFlBY0toL0wKZEFGTFNMN3NHRFFweGpleTFmWEtpeXdQUm5MZEZ2aTBac29UcXhsRUJweVBlQjlWSXZLckFuN3cyWVY0VnJIdQpsbGcwV0FmL2s3cTdsMXRGUmxnaFFSN084UVVtS1daUGNGRjhqR1lYZHZhWUdjZzRpSkM4djBBZ1VhQmxPMCt6CnEzaHBvQ0Q5NlRIUUNTdDQvT0E1Rmo2SzUweWk1N2FHd25vMTNBbVQ5eG9Qc0dqeldLMG1oUGp3M0NZQlYwbisKTit4OVBBcXBJdmhPbU1KZmpOWEQzYndoSFNjNUd4VjlpbUpBSk03MjhZazR5Nm9rVTdJNkJXZDNLV3pDaVRENwp5UHE4U1N3amxoaVk3WGtDZ2dFQkFOemp6NlFHZ1V5Zm5PZ1IwR2JEejAyMVQrOUkrdHZwSnVPQmZNWTVsTlo0CndHOGtoSkdFZk5CQUxqOXIzQXcxaDhEbEpSUmp2OVFFR2krYmFpRkJPWVB4VTlyczJnMmhIWkk5ZkdyRW1LUDMKRjJsNE1TNm1vVGx0RUxwZnM5R3FhQmxyRFlYODFYeEVSQTU0aUhQSmRoQjl0K0QxTFdxRXVLMVVaZ25NUlNQLwp2RHdZUEFXOHlvMGhiM0JDMTVvZ3BPRzMyMWl0QmpYVjhtN1pmN25rZjNyOEhWaG5BcytubW93bXg1dU8yanZ6CmtrajdxaC9WRmlHajcxc1ovV0NHQk9pQk0ySVVMNVhQUmpCM2lNQXMzNGtaTHdlOC9SL3NsaE81bzFucVhVeW4KLzV3UTFoYlNKOXpzNzdwYys0K2ROUHdJVzdHSnhEbENkQWZTd0RuNjNVRUNnZ0VBRnFhWFVZMk4yOENXZy94RwpNazJXbVhpMjIwbFh6bmpjdk9zSEJjSysrc3BaLzFJLzhOM1J1TlUzQ25UOWtpRVdwazdERUF4aFRxend0VVFFCjhJZU5CVDhJbldNMTVmVWlURWVNMDJNYTZUTUZVaFJWTnFRaVArTDJQTzN1MFI2bTdnUlZ1Z2szSTZFdHBJNEkKUUpxWitBWitVaWdGNm1Cc1RDTDR6cW5ScTZyYmZNWmFOdjNjVkhWN3NMTENkcWVncUpzdWVYdlNjeDFBUDRqZwpMWlViRFpKeFdlQ3M2d1JEQ3dvZ09COVFSWUFCNGorWW9Pb1VTNVUwaXBuYnp6eGZGZEszcWwrTWVuY3IyTkpKCldqQU4zTEl5QmZzOGxmRTZhVTZlL1NiQVFvM3RBRFJKSGUxd0tJT2UzMkxlSWljUWNqemVIK0Uza3F3YVNHVFoKWkd1U3lRS0NBUUVBbjREeGcyUWZJaEZ2NERSYzVKZ29yZGhyYkVLcXd2bk5WeU05MG5YcUFDVVo4Q2ZTZ3JIRQozeXc1T1JyTnZ4TTRnQlgzZkkyN0M0SWExcDNIT1ZROEVBYkhvcUs5b25IaFJLU1pudzl2bVpibmxRVnhubG84CnVaY0VLVkRLTEhCODB6MzJlZlprd21NWk1jbmYzcHh2WU9FblVvNDR5VjRsYlNRd3VvcUNzc2dNU09qSER1MlEKNWZCcTVBbWdYbStNSUdIL1JqMUs2cjBmWHVRMzB5Z28xY29QOXJJTDJaOFJmbnJTVUlZTEdKZDkzcTI3MzFpagpybzhPWEI2Y1ZJTHlNR0o3bENzM1lWcFhPTkJZTTAwejdXLytBZng2Vy84Zk1BY3c2ZERPcG5mNW45eVllOG90CmR0NnhEVVh2Y1hqM3RiYmpYNFEzNlpFTzhFZEMvNXNqQVFLQ0FRQkJSNHk5OTJyN05LOVZTZUh1TG1VSUZjd2kKS1dnMG0rVGk2TGxyeDFPS2k3b3cydDhubXlDQklNaDhGWTVJL0RsV1JpTTh0WWUrS2VBYTJCUFdpWmRVbUVQUgpKTHpBWVFjNXhYNVNSait3MDNHZjljdzBteFRNazNzbGxSUERYNEZ2bVpSRDkyRHBwWlFBbHdTU3haZHEyWERrCmMrZG9pVE9zTm04endNaFNXVTFiK2d6MjlabnVFamtMeU1HaXVlUll6bXNVdmdjL09KRzdSU2V5NzhmUWtZYm0KWnRIb3o3dFdJTFJxSVRYclFlZ2h6N2Jrc1lPMzM5QjE5bEI0blFPc2Y3MnBMVnMyTWdkRWlwUUp4VDlaVWRmaQpGZkQ1YzNlSkNINlNWQkRSdjFpV3Y5TzRSdWkvNThMcVRvYVE0bzdmRWxha2RoN3BrK3ZiWnRmNG41dlUKLS0tLS1FTkQgUlNBIFBSSVZBVEUgS0VZLS0tLS0K`
)

type EventCollector interface {
	// RecordStartEvent populates TelemetryRequest with the data about Command start and uploads the request to the telemetry backend
	RecordStartEvent(provider string)
	RecordEndEvent(err error)
	SetCLIData(*cobra.Command, *flags.GlobalFlags)
}

func NewDefaultCollector() *DefaultCollector {
	decodedCertificate, err := base64.RawStdEncoding.DecodeString(telemetryPrivateKey)
	if err != nil {
		panic(fmt.Errorf("failed to decode telemetry key string: %w", err))
	}

	privateKey, err := parsePrivateKey(decodedCertificate)
	if err != nil {
		panic(fmt.Errorf("failed to parse telemetry key: %w", err))
	}

	tokenGenerator, err := serviceaccount.JWTTokenGenerator("devpod-telemetry", privateKey)
	if err != nil {
		panic(fmt.Errorf("failed to create JWTTokenGenerator: %w", err))
	}

	return &DefaultCollector{
		executionID:    uuid.New().String(),
		startTime:      time.Now(),
		tokenGenerator: tokenGenerator,
	}
}

type DefaultCollector struct {
	mux       sync.Mutex
	startOnce sync.Once

	startTime   time.Time
	executionID string

	command     *cobra.Command
	globalFlags *flags.GlobalFlags
	provider    string

	tokenGenerator serviceaccount.TokenGenerator
}

func (d *DefaultCollector) SetCLIData(command *cobra.Command, globalFlags *flags.GlobalFlags) {
	d.mux.Lock()
	defer d.mux.Unlock()

	d.command = command
	d.globalFlags = globalFlags
}

func (d *DefaultCollector) RecordStartEvent(provider string) {
	d.startOnce.Do(func() {
		d.mux.Lock()
		defer d.mux.Unlock()

		d.provider = provider
		cmd := ""
		if d.command != nil {
			cmd = d.command.CommandPath()
		}

		if shouldSkipCommand(cmd) {
			return
		}

		ts := time.Now().UnixMicro()
		recordEvent(d.tokenGenerator, &types.TelemetryRequest{
			EventType: types.EventCommandStarted,
			Event: types.CMDStartedEvent{
				Timestamp:   ts,
				ExecutionID: d.executionID,
				Command:     cmd,
				Provider:    provider,
			},
			InstanceProperties: d.getInstanceProperties(d.command, d.executionID, ts),
		})
	})
}

func (d *DefaultCollector) RecordEndEvent(err error) {
	d.mux.Lock()
	defer d.mux.Unlock()

	// only record if there is a start event
	if d.provider == "" || (time.Since(d.startTime) < telemetrySendFinishedAfter && err == nil) {
		return
	}

	cmd := ""
	if d.command != nil {
		cmd = d.command.CommandPath()
	}

	if shouldSkipCommand(cmd) {
		return
	}

	cmdErr := ""
	if err != nil {
		cmdErr = err.Error()
	}

	ts := time.Now().UnixMicro()
	recordEvent(d.tokenGenerator, &types.TelemetryRequest{
		EventType: types.EventCommandFinished,
		Event: types.CMDFinishedEvent{
			Timestamp:      ts,
			ExecutionID:    d.executionID,
			Command:        cmd,
			Provider:       d.provider,
			Success:        err == nil,
			ProcessingTime: int(time.Since(d.startTime).Microseconds()),
			Errors:         cmdErr,
		},
		InstanceProperties: d.getInstanceProperties(d.command, d.executionID, ts),
	})
}

func recordEvent(tokenGenerator serviceaccount.TokenGenerator, r *types.TelemetryRequest) {
	token, err := tokenGenerator.GenerateToken(&jwt.Claims{}, &jwt.Claims{})
	if err != nil {
		log.Default.Debugf("failed to generate telemetry request signed token: %v", err)
		return
	}

	r.Token = token
	marshaled, err := json.Marshal(r)
	// handle potential Marshal errors
	if err != nil {
		log.Default.Debugf("failed to json.Marshal telemetry request: %v", err)
		return
	}

	// send the telemetry data and ignore the response
	client := http.Client{
		Timeout: telemetryRequestTimeout,
	}
	_, err = client.Post(
		telemetryEndpoint,
		"multipart/form-data",
		bytes.NewReader(marshaled),
	)
	if err != nil {
		log.Default.Debugf("error sending telemetry request: %v", err)
	}
}
