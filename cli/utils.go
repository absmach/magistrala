// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	smqsdk "github.com/absmach/magistrala/pkg/sdk"
	"github.com/fatih/color"
	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"
)

var (
	// Limit query parameter.
	Limit uint64 = 10
	// Offset query parameter.
	Offset uint64 = 0
	// Name query parameter.
	Name string = ""
	// Identity query parameter.
	Identity string = ""
	// Metadata query parameter.
	Metadata string = ""
	// Status query parameter.
	Status string = ""
	// ConfigPath config path parameter.
	ConfigPath string = ""
	// State query parameter.
	State string = ""
	// Topic query parameter.
	Topic string = ""
	// Contact query parameter.
	Contact string = ""
	// RawOutput raw output mode.
	RawOutput bool = false
	// Username query parameter.
	Username string = ""
	// FirstName query parameter.
	FirstName string = ""
	// LastName query parameter.
	LastName string = ""
)

func logJSONCmd(cmd cobra.Command, iList ...any) {
	for _, i := range iList {
		m, err := json.Marshal(i)
		if err != nil {
			logErrorCmd(cmd, err)
			return
		}

		pj, err := prettyjson.Format(m)
		if err != nil {
			logErrorCmd(cmd, err)
			return
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n\n", string(pj))
	}
}

func logUsageCmd(cmd cobra.Command, u string) {
	fmt.Fprintf(cmd.OutOrStdout(), color.YellowString("\nusage: %s\n\n"), u)
}

func logErrorCmd(cmd cobra.Command, err error) {
	boldRed := color.New(color.FgRed, color.Bold)
	boldRed.Fprintf(cmd.ErrOrStderr(), "\nerror: ")

	fmt.Fprintf(cmd.ErrOrStderr(), "%s\n\n", color.RedString(err.Error()))
}

func logOKCmd(cmd cobra.Command) {
	fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n\n", color.BlueString("ok"))
}

func logCreatedCmd(cmd cobra.Command, e string) {
	if RawOutput {
		fmt.Fprintln(cmd.OutOrStdout(), e)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), color.BlueString("\ncreated: %s\n\n"), e)
	}
}

func convertMetadata(m string) (map[string]any, error) {
	var metadata map[string]any
	if m == "" {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(Metadata), &metadata); err != nil {
		return nil, err
	}
	return nil, nil
}

const certFileMode = 0o644

func logSaveCertFiles(cmd cobra.Command, cert smqsdk.Certificate) {
	files := map[string][]byte{
		"cert.pem": []byte(cert.Certificate),
	}
	if cert.Key != "" {
		files["key.pem"] = []byte(cert.Key)
	}
	for filename, content := range files {
		if err := saveToFile(filename, content); err != nil {
			logErrorCmd(cmd, err)
			return
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Saved %s\n", filename)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\nAll certificate files have been saved successfully.\n")
}

func logSaveCAFiles(cmd cobra.Command, certBundle smqsdk.CertificateBundle) {
	files := map[string][]byte{
		"ca.crt": certBundle.Certificate,
	}
	for filename, content := range files {
		if err := saveToFile(filename, content); err != nil {
			logErrorCmd(cmd, err)
			return
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Saved %s\n", filename)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\nAll certificate files have been saved successfully.\n")
}

func logSaveCSRFiles(cmd cobra.Command, csr smqsdk.CSR) {
	files := map[string][]byte{
		"file.csr": csr.CSR,
	}
	for filename, content := range files {
		if err := saveToFile(filename, content); err != nil {
			logErrorCmd(cmd, err)
			return
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Saved %s\n", filename)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\nCSR file have been saved successfully.\n")
}

func logSaveCRLFile(cmd cobra.Command, crlBytes []byte) {
	filename := "ca.crl"
	if err := saveToFile(filename, crlBytes); err != nil {
		logErrorCmd(cmd, err)
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Saved %s\n", filename)
	fmt.Fprintf(cmd.OutOrStdout(), "\nCRL file has been saved successfully.\n")
}

func saveToFile(filename string, content []byte) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}
	filePath := filepath.Join(cwd, filename)
	if err := os.WriteFile(filePath, content, certFileMode); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}
	return nil
}
