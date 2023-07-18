package ctinfo

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/Juniper/apstra-go-sdk/apstra"
	"github.com/mitchellh/go-homedir"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const (
	sskLeyLogEnv = "SSLKEYLOGFILE"
)

func newKeyLogWriter(fileName string) (*os.File, error) {
	absPath, err := homedir.Expand(fileName)
	if err != nil {
		return nil, fmt.Errorf("error expanding home directory '%s' - %w", fileName, err)
	}

	err = os.MkdirAll(filepath.Dir(absPath), os.FileMode(0600))
	if err != nil {
		return nil, err
	}
	return os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
}

func GetCtMap(ctx context.Context, client *apstra.TwoStageL3ClosClient) (map[apstra.ObjectId]apstra.ConnectivityTemplate, error) {
	var cts []apstra.ConnectivityTemplate
	var err error

	cts, err = client.GetAllConnectivityTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting CTs - %w", err)
	}

	ctMap := make(map[apstra.ObjectId]apstra.ConnectivityTemplate, len(cts))
	for i, ct := range cts {
		if ct.Id == nil {
			return nil, fmt.Errorf("CT at index %d has nil ID", i)
		}
		if _, ok := ctMap[*ct.Id]; ok {
			return nil, fmt.Errorf("CT %q appears more than once in the CT slice", *ct.Id)
		}
		ctMap[*ct.Id] = ct
	}

	return ctMap, nil
}

func GetCtStateMap(ctx context.Context, client *apstra.TwoStageL3ClosClient) (map[apstra.ObjectId]apstra.ConnectivityTemplateState, error) {
	var ctStates []apstra.ConnectivityTemplateState
	var err error

	ctStates, err = client.GetAllConnectivityTemplateStates(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting CTs - %w", err)
	}

	ctStateMap := make(map[apstra.ObjectId]apstra.ConnectivityTemplateState, len(ctStates))
	for _, ctState := range ctStates {
		if _, ok := ctStateMap[ctState.Id]; ok {
			return nil, fmt.Errorf("CT %q appears more than once in the CT state slice", ctState.Id)
		}
		ctStateMap[ctState.Id] = ctState
	}

	return ctStateMap, nil
}

func SetupClient(ctx context.Context, cfg apstra.ClientCfg, insecure bool) (*apstra.Client, error) {
	var keyLogWriter io.Writer
	if keyLogFile, ok := os.LookupEnv(sskLeyLogEnv); ok {
		var err error
		keyLogWriter, err = newKeyLogWriter(keyLogFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	cfg.HttpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure,
				KeyLogWriter:       keyLogWriter,
			},
		},
	}

	return cfg.NewClient(ctx)
}
