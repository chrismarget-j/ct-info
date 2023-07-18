package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/Juniper/apstra-go-sdk/apstra"
	"github.com/chrismarget-j/ctinfo"
	"os"
	"path"
)

const (
	urlFlag          = "url"
	urlFlagName      = "Apstra URL"
	urlFlagEnv       = "APSTRA_URL"
	userFlag         = "user"
	userFlagName     = "Apstra username"
	userFlagEnv      = "APSTRA_USER"
	passFlag         = "pass"
	passFlagName     = "Apstra password"
	passFlagEnv      = "APSTRA_PASS"
	outDirFlag       = "out"
	outDirFlagName   = "output directory"
	outDirFlagEnv    = "OUT_DIR"
	accessBpFlag     = "access"
	accessBpFlagName = "Access Blueprint ID"
	accessBpFlagEnv  = "ACCESS_ID"
	jFrakesMsg       = "%s must not be empty. Use -%s or env var %s"

	mainBp = "42a2f25d-ba82-423c-9bf2-c715c7ce9748"
)

var apstraUrl, apstraUser, apstraPass, accessBp, outDir *string
var insecure *bool
var client *apstra.Client
var mainBpClient *apstra.TwoStageL3ClosClient
var accessBpClient *apstra.TwoStageL3ClosClient
var fw outPutter

func flagFromEnv(stringPtr *string, env string, errMsg string) error {
	if *stringPtr == "" {
		if s, ok := os.LookupEnv(env); ok {
			stringPtr = &s
		} else {
			return errors.New(errMsg)
		}
	}

	return nil
}

func config(ctx context.Context) error {
	apstraUrl = flag.String(urlFlag, "", urlFlagName)
	apstraUser = flag.String(userFlag, "admin", userFlagName)
	apstraPass = flag.String(passFlag, "", passFlagName)
	insecure = flag.Bool("insecure", false, "set to ignore TLS validation failures")
	accessBp = flag.String(accessBpFlag, "", accessBpFlagName)
	//mainBp = flag.String(mainBpFlag, "", mainBpFlagName)
	outDir = flag.String(outDirFlag, ".", outDirFlagName)
	flag.Parse()

	var err error
	err = flagFromEnv(apstraUrl, urlFlagEnv, fmt.Sprintf(jFrakesMsg, urlFlagName, urlFlag, urlFlagEnv))
	if err != nil {
		return err
	}
	err = flagFromEnv(apstraUser, userFlagEnv, fmt.Sprintf(jFrakesMsg, userFlagName, userFlag, userFlagEnv))
	if err != nil {
		return err
	}
	err = flagFromEnv(apstraPass, passFlagEnv, fmt.Sprintf(jFrakesMsg, passFlagName, passFlag, passFlagEnv))
	if err != nil {
		return err
	}
	err = flagFromEnv(accessBp, accessBpFlagEnv, fmt.Sprintf(jFrakesMsg, accessBpFlagName, accessBpFlag, accessBpFlagEnv))
	if err != nil {
		return err
	}
	err = flagFromEnv(outDir, outDirFlagEnv, fmt.Sprintf(jFrakesMsg, outDirFlagName, outDirFlag, outDirFlagEnv))
	if err != nil {
		return err
	}
	if *accessBp == mainBp {
		return fmt.Errorf("don't use the main blueprint ID for -%s s", accessBpFlag)
	}

	client, err = ctinfo.SetupClient(ctx, apstra.ClientCfg{
		Url:     *apstraUrl,
		User:    *apstraUser,
		Pass:    *apstraPass,
		Timeout: -1,
	}, *insecure)
	if err != nil {
		return err
	}

	accessBpClient, err = client.NewTwoStageL3ClosClient(ctx, apstra.ObjectId(*accessBp))
	if err != nil {
		return fmt.Errorf("error creating client for blueprint %q - %w", *accessBp, err)
	}

	mainBpClient, err = client.NewTwoStageL3ClosClient(ctx, mainBp)
	if err != nil {
		return fmt.Errorf("error creating client for blueprint %q - %w", mainBp, err)
	}

	fw = newFileWriter(path.Clean(*outDir))
	err = fw.init()
	if err != nil {
		return err
	}

	accessSingleTagged = make(map[apstra.Vlan][]apstra.ObjectId)
	accessSingleUntagged = make(map[apstra.Vlan][]apstra.ObjectId)
	mainSingleTagged = make(map[apstra.Vlan][]apstra.ObjectId)
	mainSingleUntagged = make(map[apstra.Vlan][]apstra.ObjectId)

	return nil
}
