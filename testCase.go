package main

import (
	"bufio"
	"bytes"
	"os"
)

type testCase struct {
	in  *bytes.Buffer
	out *bytes.Buffer
}

type failedTest struct {
	expected string
	got      string
}

func newTestCase(filepath string) (*testCase, error) {
	file, err := os.Open(filepath)
	defer file.Close()

	if err != nil {
		return &testCase{}, err
	}

	scanner := bufio.NewScanner(file)

	var in, out bytes.Buffer
	var builder *bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()

		switch line {
		case "$$IN:":
			builder = &in
			continue
		case "$$OUT:":
			builder = &out
			continue
		}

		if builder != nil {
			builder.WriteString(line)
			builder.WriteByte('\n')
		}

	}

	return &testCase{in: &in, out: &out}, nil
}
