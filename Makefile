.PHONY: squid copy-squid-config run

squid:
	docker container run --rm --name squid -p 3128:3128 -v $(PWD)/squid:/etc/squid/ datadog/squid

copy-squid-config:
	docker cp squid:/etc/squid/squid.conf ./squid/squid.conf

run:
	go run main.go