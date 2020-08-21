.PHONY: copy-squid-config squid run

default: run

copy-squid-config:
	docker cp squid:/etc/squid/squid.conf ./squid/squid.conf

squid:
	docker container run --rm --name squid -p 3128:3128 -v $(PWD)/squid:/etc/squid/ datadog/squid

log-squid:
	docker container exec -it squid tail -f /var/log/squid/access.log

run:
	go run main.go -addrhost 104.248.239.81