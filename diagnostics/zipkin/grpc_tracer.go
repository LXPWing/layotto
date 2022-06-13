/*
* Copyright 2021 Layotto Authors
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package zipkin

import (
	"context"
	"time"

	"mosn.io/layotto/diagnostics/grpc"

	"github.com/openzipkin/zipkin-go"
	reporterhttp "github.com/openzipkin/zipkin-go/reporter/http"
	"mosn.io/api"
	ltrace "mosn.io/layotto/components/trace"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/log"
)

const (
	service_name       = "service_name"
	reporter_endpoint  = "reporter_endpoint"
	reporter_host_post = "reporter_host_post"
	configs            = "config"

	defaultReporterEndpoint = "http://127.0.0.1:9411/api/v2/spans"
	defaultServiceName      = "layotto"
	defaultReporterHostPost = "127.0.0.1:9000"
)

type grpcZipTracer struct {
	*zipkin.Tracer
}

type grpcZipSpan struct {
	*ltrace.Span
	tracer *grpcZipTracer
	ctx    context.Context
	span   zipkin.Span
}

func NewGrpcZipTracer(traceCfg map[string]interface{}) (api.Tracer, error) {
	reporter := reporterhttp.NewReporter(getReporterEndpoint(traceCfg))
	endpoint, err := zipkin.NewEndpoint(getServerName(traceCfg), getRecorderHostPort(traceCfg))
	if err != nil {
		log.DefaultLogger.Errorf("[layotto] [zipkin] [tracer] unable to create zipkin reporter endpoint")
		return nil, err
	}

	tracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint))
	if err != nil {
		log.DefaultLogger.Errorf("[layotto] [zipkin] [tracer] cannot initialize zipkin Tracer")
		return nil, err
	}

	log.DefaultLogger.Infof("[layotto] [zipkin] [tracer] create success")

	return &grpcZipTracer{
		tracer,
	}, nil
}

func getRecorderHostPort(traceCfg map[string]interface{}) string {
	if cfg, ok := traceCfg[configs]; ok {
		recorderHostPort := cfg.(map[string]interface{})
		if point, ok := recorderHostPort[reporter_host_post]; ok {
			return point.(string)
		}
	}

	return defaultReporterHostPost
}

func getReporterEndpoint(traceCfg map[string]interface{}) string {
	if cfg, ok := traceCfg[configs]; ok {
		endpoint := cfg.(map[string]interface{})
		if point, ok := endpoint[reporter_endpoint]; ok {
			return point.(string)
		}
	}

	return defaultReporterEndpoint
}

func getServerName(traceCfg map[string]interface{}) string {
	if cfg, ok := traceCfg[configs]; ok {
		serverName := cfg.(map[string]interface{})
		if name, ok := serverName[service_name]; ok {
			return name.(string)
		}
	}

	return defaultServiceName
}

func (t *grpcZipTracer) Start(ctx context.Context, request interface{}, _ time.Time) api.Span {
	info, ok := request.(*grpc.RequestInfo)
	if !ok {
		log.DefaultLogger.Debugf("[layotto] [zipkin] [tracer] unable to get request header, downstream trace ignored")
		return nil
	}

	span := t.StartSpan(info.FullMethod)

	return &grpcZipSpan{
		tracer: t,
		ctx:    ctx,
		Span:   &ltrace.Span{},
		span:   span,
	}
}

func (s *grpcZipSpan) TraceId() string {
	return s.span.Context().TraceID.String()
}

func (s *grpcZipSpan) InjectContext(requestHeaders types.HeaderMap, requestInfo api.RequestInfo) {
}

func (s *grpcZipSpan) SetRequestInfo(requestInfo api.RequestInfo) {
}

func (s *grpcZipSpan) FinishSpan() {
	s.span.Finish()
}
