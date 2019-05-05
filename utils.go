package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"
)

func errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(-1)
}

func runProgram(program string, inBuffer *bytes.Buffer, timeout time.Duration, cmdArgs ...string) (*bytes.Buffer, error) {
	execCmd := exec.Command(program, cmdArgs...)

	if inBuffer != nil {
		execCmd.Stdin = inBuffer
	}

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	execCmd.Stdout = &outBuf
	execCmd.Stderr = &errBuf

	if err := execCmd.Start(); err != nil {
		return nil, err
	}

	isRunning := true
	timeoutReached := false
	if timeout != 0 {
		go func() {
			start := time.Now()

			for isRunning {
				if time.Since(start) > timeout {
					execCmd.Process.Kill()
					timeoutReached = true
					break
				}

				time.Sleep(250 * time.Millisecond) // Check every 250ms
			}
		}()
	}

	if err := execCmd.Wait(); err != nil {
		if timeoutReached {
			return nil, fmt.Errorf("program took more than %f seconds to execute", timeout.Seconds())
		}
		return nil, fmt.Errorf("%v\nstderr: %s", err, errBuf.String())
	}

	isRunning = false

	return &outBuf, nil
}

func compileJava(outDir string, release int, files []string) error {
	args := []string{"--release", strconv.Itoa(release), "-d", outDir} // Ugly

	_, err := runProgram("javac", nil, 0, append(args, files...)...)

	return err
}

// Run a java program with 3 seconds of timeout
func runJava(binDir, srcdir, entry string, in *bytes.Buffer) (*bytes.Buffer, error) {
	return runProgram("java", in, time.Duration(3*time.Second), "-cp", binDir, path.Join(srcdir, entry))
}
