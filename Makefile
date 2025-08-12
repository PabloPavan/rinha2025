SERVICE ?=
COMPOSE ?=
PPROF_HOST ?= localhost:6060
PPROF_SECS ?= 60

DOCKER_COMPOSE_CMD := docker compose $(if $(COMPOSE),-f $(COMPOSE),)

restart:
	@echo "Reiniciando serviço(s) $(SERVICE)"
	$(DOCKER_COMPOSE_CMD) restart $(SERVICE)

rebuild:
	@echo "Rebuildando e reiniciando serviço(s) $(SERVICE)"
	$(DOCKER_COMPOSE_CMD)  down --remove-orphans $(SERVICE)
	$(DOCKER_COMPOSE_CMD)  up --build --force-recreate --remove-orphans $(SERVICE)

pprof-cpu:
	go tool pprof -http=:0 http://$(PPROF_HOST)/debug/pprof/profile?seconds=$(PPROF_SECS)

pprof-heap:
	go tool pprof -http=:0 http://$(PPROF_HOST)/debug/pprof/heap

pprof-trace:
	curl -o trace.out "http://$(PPROF_HOST)/debug/pprof/trace?seconds=$(PPROF_SECS)"
	go tool trace trace.out