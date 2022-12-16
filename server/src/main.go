// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrgin/v1"
)

func makeGinEndpoint(s string) func(*gin.Context) {
	return func(c *gin.Context) {
		c.Writer.WriteString(s)
	}
}

func v1login(c *gin.Context)  { c.Writer.WriteString("v1 login") }
func v1submit(c *gin.Context) { c.Writer.WriteString("v1 submit") }

func endpointPanic(c *gin.Context) {
	panic("v1 panic")
	c.Writer.WriteHeader(http.StatusInternalServerError)
	c.Writer.WriteString("v1 panic")
}

func endpoint404(c *gin.Context) {
	c.Writer.WriteHeader(404)
	c.Writer.WriteString("returning 404")
}

func endpoint500(c *gin.Context) {
	c.Writer.WriteHeader(http.StatusInternalServerError)
	c.Writer.WriteString("returning 500")
}

func endpointResponseHeaders(c *gin.Context) {
	c.Writer.WriteHeader(200)
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteString(`{"zip":"zap"}`)
}

func endpointNotFound(c *gin.Context) {
	c.Writer.WriteString("there's no endpoint for that!")
}

func endpointAccessTransaction(c *gin.Context) {
	if txn := nrgin.Transaction(c); nil != txn {
		txn.SetName("custom-name")
	}
	c.Writer.WriteString("changed the name of the transaction!")
}

// endpointStressCpu はcpuに負荷をかけるエンドポイント
// 参照 https://stackoverflow.com/questions/41079492/how-to-artificially-increase-cpu-usage
func endpointStressCpu(c *gin.Context) {
	done := make(chan int)

	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
				}
			}
		}()
	}

	time.Sleep(time.Second * 10)
	close(done)

	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.WriteString("cpu stress test")
}

func endpointStressMemory(c *gin.Context) {
	buf := bytes.Buffer{}

	for i := 0; i < 10000; i++ {
		_, err := buf.WriteString(strings.Repeat("test", 10000))
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Millisecond)
	}

	fmt.Println(buf.String())
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.WriteString("memory stress test2")
}

func mustGetEnv(key string) string {
	if val := os.Getenv(key); "" != val {
		return val
	}
	panic(fmt.Sprintf("environment variable %s unset", key))
}

func main() {
	cfg := newrelic.NewConfig("new relic docker", "xxxxxxxxxxxxxxxx")
	cfg.Logger = newrelic.NewDebugLogger(os.Stdout)
	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	router := gin.Default()
	router.Use(nrgin.Middleware(app))

	router.GET("/404", endpoint404)
	router.GET("/500", endpoint500)
	router.GET("/panic", endpointPanic)
	router.GET("/headers", endpointResponseHeaders)
	router.GET("/txn", endpointAccessTransaction)
	router.GET("/stress/cpu", endpointStressCpu)
	router.GET("/stress/memory", endpointStressMemory)

	// Since the handler function name is used as the transaction name,
	// anonymous functions do not get usefully named.  We encourage
	// transforming anonymous functions into named functions.
	router.GET("/anon", func(c *gin.Context) {
		c.Writer.WriteString("anonymous function handler")
	})

	v1 := router.Group("/v1")
	v1.GET("/login", v1login)
	v1.GET("/submit", v1submit)

	router.NoRoute(endpointNotFound)

	err = router.Run(":8080")
	if err != nil {
		panic(err)
	}
}
