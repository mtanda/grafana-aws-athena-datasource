module github.com/mtanda/grafana-aws-athena-datasource

go 1.12

require (
	github.com/aws/aws-sdk-go v1.19.37
	github.com/golang/protobuf v1.3.1
	github.com/grafana/grafana v6.0.1+incompatible
	github.com/grafana/grafana_plugin_model v0.0.0-20180518082423-84176c64269d
	github.com/hashicorp/go-hclog v0.8.0
	github.com/hashicorp/go-plugin v0.0.0-20180331002553-e8d22c780116
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af
	github.com/mitchellh/go-testing-interface v1.0.0
	github.com/oklog/run v1.0.0
	golang.org/x/net v0.0.0-20190301231341-16b79f2e4e95
	golang.org/x/sys v0.0.0-20190306220723-b294cbcfc56d
	golang.org/x/text v0.3.0
	google.golang.org/genproto v0.0.0-20190306222511-6e86cb5d2f12
	google.golang.org/grpc v1.19.0
)
