SERVICE ?=
COMPOSE ?=

DOCKER_COMPOSE_CMD := docker compose $(if $(COMPOSE),-f $(COMPOSE),)

restart:
	@echo "Reiniciando serviço(s) $(SERVICE)"
	$(DOCKER_COMPOSE_CMD) restart $(SERVICE)

rebuild:
	@echo "Rebuildando e reiniciando serviço(s) $(SERVICE)"
	$(DOCKER_COMPOSE_CMD) up -d --build $(SERVICE)