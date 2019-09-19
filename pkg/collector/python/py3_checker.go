// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

// +build python

package python

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
)

var (
	linterTimeout = time.Duration(config.Datadog.GetInt("python3_linter_timeout")) * time.Second
)

type warning struct {
	Message string
}

// validatePython3 checks that a check can run on python 3.
func validatePython3(moduleName string, modulePath string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), linterTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, pythonBinPath, "-m", "a7", modulePath)

	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error running the linter on (%s): %s", err, stderr.String())
	}

	var warnings []warning
	if err := json.Unmarshal(stdout.Bytes(), &warnings); err != nil {
		return nil, fmt.Errorf("could not Unmarshal warnings from Python3 linter: %s", err)
	}

	res := []string{}
	// no post processing needed for now, we just retrieve every messages
	for _, warn := range warnings {
		res = append(res, warn.Message)
	}

	return res, nil
}
