package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

func main() {
	bindirPtr := flag.String("o", "bin", "Output directory for the class files.")
	releasePtr := flag.Int("r", 8, "Release version for the java compiler.")
	testdirPtr := flag.String("t", "tests", "Test files directory.")
	srcdirPtr := flag.String("s", "src", "Source path for java source files.")
	entryPtr := flag.String("m", "Main", "Entry point of the java program.")
	concurrencyPtr := flag.Int("c", 1, "Number of concurrent test tasks.")
	flag.Parse()

	dirfiles, err := ioutil.ReadDir(*srcdirPtr)
	if err != nil {
		errorf("Error while looking for java files in %s: %v", *srcdirPtr, err)
	}

	var javaFiles []string
	for _, file := range dirfiles {
		fName := file.Name()
		if !strings.Contains(fName, ".java") {
			continue
		}
		javaFiles = append(javaFiles, path.Join(*srcdirPtr, fName))
	}

	if len(javaFiles) == 0 {
		errorf("Did not find any java files in %s", *srcdirPtr)
	}

	fmt.Println("Compiling files: ", strings.Join(javaFiles, " "))
	if err := compileJava(*bindirPtr, *releasePtr, javaFiles); err != nil {
		errorf("Error while compiling program: %v", err)
	}

	tests := make(chan *testCase)
	// Doesn't need to be synced because it is only used after all the routines have finished
	testCounter := 0

	// Read tests
	go func() {
		testFiles, err := ioutil.ReadDir(*testdirPtr)
		if err != nil {
			errorf("Error while reading test files: %v", err)
		}

		for _, testFile := range testFiles {
			fName := testFile.Name()
			if !strings.Contains(fName, ".javatest") {
				continue
			}

			newTest, err := newTestCase(path.Join(*testdirPtr, fName))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while trying to read testfile %s", fName)
				continue
			}

			testCounter++
			tests <- newTest
		}

		close(tests)
	}()

	var wg sync.WaitGroup
	// Make a buffered channel so that if two or more routines finish their test
	// they won't block
	failedTestsChan := make(chan failedTest, 5)
	var failedTests []failedTest
	ready := make(chan bool, 1)

	// Collect the channel into a slice
	go func() {
		for f := range failedTestsChan {
			failedTests = append(failedTests, f)
		}
		ready <- true
	}()

	wg.Add(*concurrencyPtr)
	start := time.Now()

	for i := 0; i < *concurrencyPtr; i++ {
		go func() {
			defer wg.Done()
			for test := range tests {
				out, err := runJava(*bindirPtr, *entryPtr, test.in)

				sTestOut := strings.TrimSpace(test.out.String())
				if err != nil {
					failedTestsChan <- failedTest{expected: sTestOut, got: err.Error()}
					continue
				}

				sOut := strings.TrimSpace(out.String())
				if strings.Compare(sOut, sTestOut) != 0 {
					failedTestsChan <- failedTest{expected: sTestOut, got: sOut}
				}
			}
		}()
	}

	wg.Wait()
	close(failedTestsChan)
	<-ready // Wait for failedTestsChan to drain

	lFailed := len(failedTests)

	failClr := color.New(color.Bold, color.FgWhite, color.BgRed).SprintfFunc()
	fmt.Printf("Successful Tests (%d/%d)\n", testCounter-lFailed, testCounter)

	if lFailed > 0 {
		fmt.Println("\nFailed Tests: ")
		for i, f := range failedTests {
			fmt.Println(failClr("Test (%d)", i))
			fmt.Println("\tExpected:")
			fmt.Printf("\t%s\n", f.expected)
			fmt.Println("\tGot:")
			fmt.Printf("\t%s\n", f.got)
			fmt.Println()
		}
	}

	fmt.Printf("Finished in %v", time.Since(start))
}
