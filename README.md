# grafana aws athena datasource

Grafana plugin for queryng AWS Athena as data source

## Installation

```sh
docker run -d --name=grafana -p 3000:3000 -e GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=mtanda-aws-athena-datasource grafana/grafana

docker exec -it grafana /bin/bash
# inside container
grafana-cli --pluginUrl https://github.com/mtanda/grafana-aws-athena-datasource/releases/download/2.2.7/grafana-aws-athena-datasource-2.2.7.zip plugins install grafana-aws-athena-datasource
exit

# outside container
docker restart grafana
```
