// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package transport

import (
	"context"
	"errors"
	"fmt"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/prometheus/storage/remote"
	"net/http"

	"go.opentelemetry.io/collector/consumer"
)

var errNilListenAndServeParameters = errors.New("no parameter of ListenAndServe can be nil")

// Server abstracts the type of transport being used and offer an
// interface to handle serving clients over that transport.
type Server interface {
	// ListenAndServe is a blocking call that starts to listen for client messages
	// on the specific transport, and prepares the message to be processed by
	// the Parser and passed to the next consumer.
	ListenAndServe(
		mc consumer.Metrics,
		sc consumer.Logs,
		tc consumer.Traces,
		r Reporter,
		ts chan<- prompb.TimeSeries,
		samples chan<- prompb.Sample,
		exemplars chan<- prompb.Exemplar,
		histograms chan<- prompb.Histogram,
	) error

	// Close stops any running ListenAndServe, however, it waits for any
	// data already received to be parsed and sent to the next consumer.
	Close() error
}

func (*Server) ListenAndServe() {}

// Reporter is used to report (via zPages, logs, metrics, etc) the events
// happening when the Server is receiving and processing data.
type Reporter interface {
	// OnDataReceived is called when a message or request is received from
	// a client. The returned context should be used in other calls to the same
	// reporter instance. The caller code should include a call to end the
	// returned span.
	OnDataReceived(ctx context.Context) context.Context

	// OnTranslationError is used to report a translation error from original
	// format to the internal format of the Collector. The context
	// passed to it should be the ones returned by OnDataReceived.
	OnTranslationError(ctx context.Context, err error)

	// OnMetricsProcessed is called when the received data is passed to next
	// consumer on the pipeline. The context passed to it should be the
	// one returned by OnDataReceived. The error should be error returned by
	// the next consumer - the reporter is expected to handle nil error too.
	OnMetricsProcessed(ctx context.Context, numReceivedMessages int, err error)

	// OnDebugf allows less structured reporting for debugging scenarios.
	OnDebugf(
		template string,
		args ...interface{})
}

func handleRemoteWrite(w http.ResponseWriter, r *http.Request) {
	req, err := remote.DecodeWriteRequest(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// TODO hughesjj figure out how to apply backpressure
	//// Check if the work queue is full
	//if len(workQueue) >= maxQueueSize {
	//	w.Header().Set("Retry-After", "30") // Suggest a 30-second delay before the next request
	//	http.Error(w, "Work queue is full, apply backpressure", http.StatusTooManyRequests)
	//	return
	//}

	for _, ts := range req.Timeseries {
		m := make(model.Metric, len(ts.Labels))
		for _, l := range ts.Labels {
			m[model.LabelName(l.Name)] = model.LabelValue(l.Value)
		}
		// workQueue <- ts
		fmt.Println(m)

		for _, s := range ts.Samples {
			// TODO hughesjj enque sample?
			fmt.Printf("\tSample:  %f %d\n", s.Value, s.Timestamp)
		}

		for _, e := range ts.Exemplars {
			m := make(model.Metric, len(e.Labels))
			for _, l := range e.Labels {
				m[model.LabelName(l.Name)] = model.LabelValue(l.Value)
			}
			// TODO hughesjj enque exemplar?
			fmt.Printf("\tExemplar:  %+v %f %d\n", m, e.Value, e.Timestamp)
		}

		for _, hp := range ts.Histograms {
			h := remote.HistogramProtoToHistogram(hp)
			// TODO hughesjj enque histogram?
			fmt.Printf("\tHistogram:  %s\n", h.String())
		}
	}

	// In anticipation of eventually better supporting backpressure, return 202 instead of 204
	w.WriteHeader(http.StatusAccepted)
}
