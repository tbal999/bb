package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func captureOutput(f func()) string {
	var buf bytes.Buffer
	log.SetFlags(0)
	log.SetOutput(&buf)
	f()
	log.SetOutput(os.Stderr)
	return buf.String()
}

func TestMain(t *testing.T) {
	admin = "root" // for docker!
	tests := []struct {
		name           string
		action         func()
		expectedOutput string
	}{
		{
			name: "happy: type q to quit",
			action: func() {
				var offset int64 = 0
				input, err := ioutil.TempFile("", "")
				if err != nil {
					t.Fatal(err)
				}
				defer input.Close()
				_, err = io.WriteString(input, "q")
				if err != nil {
					t.Fatal(err)
				}
				_, err = input.Seek(offset, 0)
				if err != nil {
					t.Fatal(err)
				}
				initiateBB([]string{" "})
				ViewBB(" ", input, 500)
			},
			expectedOutput: "",
		},
		{
			name: "happy: type h for help",
			action: func() {
				var wg sync.WaitGroup
				var offset int64 = 0
				input, err := ioutil.TempFile("", "")
				if err != nil {
					t.Fatal(err)
				}
				defer input.Close()
				_, err = io.WriteString(input, "h")
				if err != nil {
					t.Fatal(err)
				}
				_, err = input.Seek(offset, 0)
				if err != nil {
					t.Fatal(err)
				}
				wg.Add(1)
				go func() {
					defer wg.Done()
					ViewBB(" ", input, 500)
				}()
				offset++
				wg.Add(1)
				go func() {
					defer wg.Done()
					time.Sleep(1 * time.Duration(time.Second))
					_, err = io.WriteString(input, "!")
					if err != nil {
						t.Fatal(err)
					}
					_, err = input.Seek(offset, 0)
					if err != nil {
						t.Fatal(err)
					}
				}()
				offset++
				go func() {
					defer wg.Done()
					time.Sleep(1 * time.Duration(time.Second))
					_, err = io.WriteString(input, "q")
					if err != nil {
						t.Fatal(err)
					}
					_, err = input.Seek(offset, 0)
					if err != nil {
						t.Fatal(err)
					}
				}()
				offset++
				wg.Wait()
			},
			expectedOutput: "",
		},
		/*{
			name: "happy: create a board called 'hello'",
			action: func() {
				var wg sync.WaitGroup
				var offset int64 = 0
				input, err := ioutil.TempFile("", "")
				if err != nil {
					t.Fatal(err)
				}
				defer input.Close()
				_, err = io.WriteString(input, "new hello")
				if err != nil {
					t.Fatal(err)
				}
				_, err = input.Seek(offset, 0)
				if err != nil {
					t.Fatal(err)
				}
				wg.Add(1)
				go func() {
					defer wg.Done()
					ViewBB(" ", input, 500)
				}()
				offset += 9
				wg.Add(1)
				go func() {
					defer wg.Done()
					time.Sleep(1 * time.Duration(time.Second))
					_, err = io.WriteString(input, "1")
					if err != nil {
						t.Fatal(err)
					}
					_, err = input.Seek(offset, 0)
					if err != nil {
						t.Fatal(err)
					}
					offset += 1

					time.Sleep(1 * time.Duration(time.Second))
					_, err = io.WriteString(input, "hey")
					if err != nil {
						t.Fatal(err)
					}
					_, err = input.Seek(offset, 0)
					if err != nil {
						t.Fatal(err)
					}
					offset += 3

					time.Sleep(1 * time.Duration(time.Second))
					_, err = io.WriteString(input, "heyy")
					if err != nil {
						t.Fatal(err)
					}
					_, err = input.Seek(offset, 0)
					if err != nil {
						t.Fatal(err)
					}
					offset += 4

					time.Sleep(1 * time.Duration(time.Second))
					_, err = io.WriteString(input, "q")
					if err != nil {
						t.Fatal(err)
					}
					_, err = input.Seek(offset, 0)
					if err != nil {
						t.Fatal(err)
					}
					offset += 1
				}()
				wg.Wait()
			},
			expectedOutput: "",
		},*/
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			actualOutput := captureOutput(tt.action)
			log.Println("_______________")
			time.Sleep(time.Duration(1) * time.Second)
			assert.Equal(t, tt.expectedOutput, actualOutput, tt.name)
		})
	}

}
