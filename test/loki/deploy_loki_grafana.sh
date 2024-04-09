#!/bin/bash

sudo apt-get install -y adduser libfontconfig1 musl
wget https://dl.grafana.com/enterprise/release/grafana-enterprise_10.4.2_amd64.deb
sudo dpkg -i grafana-enterprise_10.4.2_amd64.deb
sudo /bin/systemctl start grafana-server

curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

docker run -d --name loki -p 3100:3100 grafana/loki:latest

#docker run -d --name grafana -p 3000:3000 grafana/grafana:latest

sleep 10

curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Loki",
    "type": "loki",
    "access": "proxy",
    "url": "http://localhost:3100",
    "jsonData": {
      "httpMethod": "POST"
    }
  }' \
  http://localhost:3000/api/datasources

curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "dashboard": {
      "id": null,
      "uid": null,
      "title": "Loki Logs",
      "panels": [],
      "templating": {
        "list": []
      },
      "annotations": {
        "list": []
      },
      "editable": true,
      "schemaVersion": 16,
      "version": 0,
      "links": [],
      "gnetId": null
    },
    "folderId": 0,
    "overwrite": false
  }' \
  http://localhost:3000/api/dashboards/import

echo "Loki and Grafana deployment completed."
