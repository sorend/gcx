package query

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/grafana/gcx/internal/agent"
	"github.com/grafana/gcx/internal/format"
	"github.com/grafana/gcx/internal/graph"
	cmdio "github.com/grafana/gcx/internal/output"
	"github.com/grafana/gcx/internal/query/athena"
	"github.com/grafana/gcx/internal/query/azuremonitor"
	"github.com/grafana/gcx/internal/query/bigquery"
	"github.com/grafana/gcx/internal/query/clickhouse"
	"github.com/grafana/gcx/internal/query/cloudmonitoring"
	"github.com/grafana/gcx/internal/query/cloudwatch"
	"github.com/grafana/gcx/internal/query/elasticsearch"
	"github.com/grafana/gcx/internal/query/infinity"
	"github.com/grafana/gcx/internal/query/influxdb"
	"github.com/grafana/gcx/internal/query/loki"
	"github.com/grafana/gcx/internal/query/opensearch"
	"github.com/grafana/gcx/internal/query/prometheus"
	"github.com/grafana/gcx/internal/query/pyroscope"
	querysql "github.com/grafana/gcx/internal/query/sql"
	"github.com/grafana/gcx/internal/query/tempo"
	"github.com/grafana/gcx/internal/terminal"
)

type queryTableCodec struct{}

func (c *queryTableCodec) Format() format.Format {
	return "table"
}

func (c *queryTableCodec) Encode(w io.Writer, data any) error {
	switch resp := data.(type) {
	case *prometheus.QueryResponse:
		return prometheus.FormatTable(w, resp)
	case *loki.QueryResponse:
		return loki.FormatQueryTable(w, resp)
	case *loki.MetricQueryResponse:
		return loki.FormatMetricQueryTable(w, resp)
	case *loki.PatternsResponse:
		return loki.FormatPatternsTable(w, resp)
	case *pyroscope.QueryResponse:
		return pyroscope.FormatQueryTable(w, resp)
	case *tempo.SearchResponse:
		return tempo.FormatSearchTable(w, resp)
	case *tempo.BaselineResult:
		return tempo.FormatBaselineTable(w, resp)
	case *tempo.MetricsResponse:
		return tempo.FormatMetricsTable(w, resp)
	case *infinity.QueryResponse:
		return infinity.FormatTable(w, resp)
	case *influxdb.QueryResponse:
		return influxdb.FormatQueryTable(w, resp)
	case *tempo.GetTraceResponse:
		return tempo.FormatTraceTable(w, resp)
	case *querysql.QueryResponse:
		return querysql.FormatTable(w, resp)
	case []clickhouse.TableInfo:
		return clickhouse.FormatListTablesTable(w, resp)
	case []clickhouse.ColumnInfo:
		return clickhouse.FormatDescribeTableTable(w, resp)
	case athena.StringList:
		return athena.FormatStringList(w, resp.Items, resp.Header)
	case bigquery.StringList:
		return bigquery.FormatStringList(w, resp.Items, resp.Header)
	case []bigquery.TableInfo:
		return bigquery.FormatListTablesTable(w, resp)
	case []bigquery.ColumnInfo:
		return bigquery.FormatDescribeTableTable(w, resp)
	case *cloudwatch.QueryResponse:
		return cloudwatch.FormatTable(w, resp)
	case *cloudmonitoring.QueryResponse:
		return cloudmonitoring.FormatTable(w, resp)
	case *elasticsearch.MetricsResponse:
		return elasticsearch.FormatMetricsTable(w, resp)
	case *opensearch.MetricsResponse:
		return opensearch.FormatMetricsTable(w, resp)
	case *azuremonitor.QueryResponse:
		return azuremonitor.FormatTable(w, resp)
	case *azuremonitor.TableResponse:
		return azuremonitor.FormatTableResponse(w, resp)
	default:
		return errors.New("invalid data type for query table codec")
	}
}

func (c *queryTableCodec) Decode(io.Reader, any) error {
	return errors.New("query table codec does not support decoding")
}

type queryWideCodec struct{}

func (c *queryWideCodec) Format() format.Format {
	return "wide"
}

func (c *queryWideCodec) Encode(w io.Writer, data any) error {
	switch resp := data.(type) {
	case *prometheus.QueryResponse:
		return prometheus.FormatWideTable(w, resp)
	case *loki.QueryResponse:
		return loki.FormatQueryTableWide(w, resp)
	case *loki.PatternsResponse:
		return loki.FormatPatternsTable(w, resp)
	case *tempo.SearchResponse:
		return tempo.FormatSearchTable(w, resp)
	case *tempo.BaselineResult:
		return tempo.FormatBaselineTable(w, resp)
	case *infinity.QueryResponse:
		return infinity.FormatTable(w, resp)
	case *tempo.GetTraceResponse:
		return tempo.FormatTraceWide(w, resp)
	case *querysql.QueryResponse:
		return querysql.FormatWideTable(w, resp)
	case athena.StringList:
		return athena.FormatStringList(w, resp.Items, resp.Header)
	case bigquery.StringList:
		return bigquery.FormatStringList(w, resp.Items, resp.Header)
	case []bigquery.TableInfo:
		return bigquery.FormatListTablesTable(w, resp)
	case []bigquery.ColumnInfo:
		return bigquery.FormatDescribeTableTable(w, resp)
	case *cloudwatch.QueryResponse:
		return cloudwatch.FormatWide(w, resp)
	case *cloudmonitoring.QueryResponse:
		return cloudmonitoring.FormatWide(w, resp)
	case *elasticsearch.MetricsResponse:
		return elasticsearch.FormatMetricsTable(w, resp)
	case *opensearch.MetricsResponse:
		return opensearch.FormatMetricsTable(w, resp)
	case *azuremonitor.QueryResponse:
		return azuremonitor.FormatWide(w, resp)
	case *azuremonitor.TableResponse:
		return azuremonitor.FormatTableResponse(w, resp)
	default:
		return errors.New("invalid data type for query wide codec")
	}
}

func (c *queryWideCodec) Decode(io.Reader, any) error {
	return errors.New("query wide codec does not support decoding")
}

type queryGraphCodec struct{}

func (c *queryGraphCodec) Format() format.Format {
	return "graph"
}

func (c *queryGraphCodec) Encode(w io.Writer, data any) error {
	chartData, err := chartDataFor("graph", data)
	if err != nil {
		return err
	}
	return graph.RenderChart(w, chartData, graph.DefaultChartOptions())
}

func (c *queryGraphCodec) Decode(io.Reader, any) error {
	return errors.New("graph codec does not support decoding")
}

// chartDataFor converts a datasource query response into chart data for the
// named chart format ("graph" or "sixel"). Errors name that format.
func chartDataFor(formatName string, data any) (*graph.ChartData, error) {
	var chartData *graph.ChartData
	var err error

	// Each success case only assigns chartData/err; the single check below
	// the switch (rather than one per case) is what keeps this dispatch
	// table's complexity flat as datasource kinds are added — a repeated
	// "if err != nil { return err }" per case was pushing gocyclo over its
	// threshold on every new kind, when the cases never disagree on how to
	// react to an error.
	switch resp := data.(type) {
	case *prometheus.QueryResponse:
		chartData, err = graph.FromPrometheusResponse(resp)
	case *loki.QueryResponse:
		chartData, err = graph.FromLokiLogVolumeResponse(resp)
	case *loki.MetricQueryResponse:
		chartData, err = graph.FromLokiMetricResponse(resp)
	case *pyroscope.QueryResponse:
		chartData, err = graph.FromPyroscopeResponse(resp)
	case *cloudwatch.QueryResponse:
		chartData, err = graph.FromCloudWatchResponse(resp)
	case *cloudmonitoring.QueryResponse:
		chartData, err = graph.FromCloudMonitoringResponse(resp)
	case *azuremonitor.QueryResponse:
		chartData, err = graph.FromAzureMonitorResponse(resp)
	case *azuremonitor.TableResponse:
		return nil, unsupportedChartData(formatName, "KQL table results")
	case *tempo.SearchResponse:
		return nil, unsupportedChartData(formatName, "trace search results")
	case *infinity.QueryResponse:
		return nil, unsupportedChartData(formatName, "Infinity queries")
	case *tempo.MetricsResponse:
		chartData, err = graph.FromTempoMetricsResponse(resp)
	case *influxdb.QueryResponse:
		chartData, err = graph.FromInfluxDBResponse(resp)
	case *elasticsearch.MetricsResponse:
		chartData, err = graph.FromElasticsearchResponse(resp)
	case *opensearch.MetricsResponse:
		chartData, err = graph.FromOpenSearchResponse(resp)
	case *querysql.QueryResponse:
		return nil, unsupportedChartData(formatName, "SQL datasource queries")
	case []clickhouse.TableInfo:
		return nil, unsupportedChartData(formatName, "ClickHouse list-tables")
	case []clickhouse.ColumnInfo:
		return nil, unsupportedChartData(formatName, "ClickHouse describe-table")
	case athena.StringList:
		return nil, unsupportedChartData(formatName, "Athena discovery")
	case bigquery.StringList:
		return nil, unsupportedChartData(formatName, "BigQuery list-datasets")
	case []bigquery.TableInfo:
		return nil, unsupportedChartData(formatName, "BigQuery list-tables")
	case []bigquery.ColumnInfo:
		return nil, unsupportedChartData(formatName, "BigQuery describe-table")
	default:
		return nil, fmt.Errorf("invalid data type for %s codec", formatName)
	}
	if err != nil {
		return nil, err
	}
	return chartData, nil
}

func unsupportedChartData(formatName, source string) error {
	return fmt.Errorf("%s output is not supported for %s; use -o table/json/yaml", formatName, source)
}

// querySixelCodec renders the same charts as the graph codec as Sixel
// graphics. It is only usable on an interactive terminal that advertises Sixel.
type querySixelCodec struct{}

// sixelProbeTimeout bounds the wait for the terminal's capability reply.
// Remote terminals (e.g. over SSH) can be slow to answer, so this is generous.
const sixelProbeTimeout = time.Second

func (c *querySixelCodec) Format() format.Format {
	return "sixel"
}

func (c *querySixelCodec) Encode(w io.Writer, data any) error {
	chartData, err := chartDataFor("sixel", data)
	if err != nil {
		return err
	}
	if agent.IsAgentMode() {
		return errors.New("sixel output is not available in agent mode; use -o json, yaml or agents")
	}
	if !terminal.StdoutIsTerminal() {
		return errors.New("sixel output requires an interactive terminal; use -o graph, json or yaml when piping")
	}
	if !terminal.SixelSupported(sixelProbeTimeout) {
		return errors.New("this terminal does not advertise Sixel graphics support; use -o graph instead")
	}
	return graph.RenderSixel(w, chartData, graph.DefaultSixelOptions())
}

func (c *querySixelCodec) Decode(io.Reader, any) error {
	return errors.New("sixel codec does not support decoding")
}

type queryJSONCodec struct {
	inner *format.JSONCodec
}

func (c *queryJSONCodec) Format() format.Format {
	return format.JSON
}

func (c *queryJSONCodec) Encode(w io.Writer, data any) error {
	// InfluxDB responses carry millisecond-epoch timestamps in time columns.
	// FormatQueryJSON converts those to RFC3339 strings so the JSON output
	// matches what users expect rather than raw numeric epoch values.
	if resp, ok := data.(*influxdb.QueryResponse); ok {
		return c.inner.Encode(w, influxdb.FormatQueryJSON(resp))
	}
	return c.inner.Encode(w, data)
}

func (c *queryJSONCodec) Decode(r io.Reader, v any) error {
	return c.inner.Decode(r, v)
}

type queryYAMLCodec struct {
	inner *format.YAMLCodec
}

func (c *queryYAMLCodec) Format() format.Format {
	return format.YAML
}

func (c *queryYAMLCodec) Encode(w io.Writer, data any) error {
	// Same as the JSON codec: convert millisecond-epoch time columns to
	// RFC3339 strings before serializing so the output is human-readable.
	if resp, ok := data.(*influxdb.QueryResponse); ok {
		return c.inner.Encode(w, influxdb.FormatQueryJSON(resp))
	}
	return c.inner.Encode(w, data)
}

func (c *queryYAMLCodec) Decode(r io.Reader, v any) error {
	return c.inner.Decode(r, v)
}

// RegisterCodecs registers the table and wide codecs, plus graph and sixel when
// enabled, on the given IO options.
func RegisterCodecs(ioOpts *cmdio.Options, enableGraph bool) {
	ioOpts.RegisterCustomCodec("table", &queryTableCodec{})
	ioOpts.RegisterCustomCodec("wide", &queryWideCodec{})
	ioOpts.RegisterCustomCodec("json", &queryJSONCodec{inner: format.NewJSONCodec()})
	ioOpts.RegisterCustomCodec("yaml", &queryYAMLCodec{inner: format.NewYAMLCodec()})
	if enableGraph {
		ioOpts.RegisterCustomCodec("graph", &queryGraphCodec{})
		ioOpts.RegisterCustomCodec("sixel", &querySixelCodec{})
	}
	ioOpts.DefaultFormat("table")
}

// RegisterStructuredCodecs registers only the JSON and YAML codecs (default
// JSON) — no table, wide, or graph. Use it for datasource commands whose
// payload is an opaque or free-form structure with no meaningful tabular
// projection, e.g. the experimental Tempo trace-diff patch. The table/wide
// codecs switch on concrete response types and reject anything else with
// "invalid data type", so advertising them for such commands would surface a
// runtime error on a documented output path.
func RegisterStructuredCodecs(ioOpts *cmdio.Options) {
	ioOpts.RegisterCustomCodec("json", &queryJSONCodec{inner: format.NewJSONCodec()})
	ioOpts.RegisterCustomCodec("yaml", &queryYAMLCodec{inner: format.NewYAMLCodec()})
	ioOpts.DefaultFormat("json")
}
