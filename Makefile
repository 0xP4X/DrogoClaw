.PHONY: help install build dev test lint format clean setup gateway cli deploy

help:
	@echo "🐉 DrogonClaw - Available Commands"
	@echo ""
	@echo "Setup:"
	@echo "  make install    Install dependencies"
	@echo "  make setup      Initialize project structure"
	@echo ""
	@echo "Development:"
	@echo "  make dev        Start development mode (watch)"
	@echo "  make build      Build TypeScript"
	@echo "  make rebuild    Clean and rebuild"
	@echo ""
	@echo "Testing:"
	@echo "  make test       Run tests"
	@echo "  make test-watch Run tests in watch mode"
	@echo "  make coverage   Generate coverage report"
	@echo ""
	@echo "Code Quality:"
	@echo "  make lint       Check code style"
	@echo "  make format     Auto-format code"
	@echo "  make clean      Remove build artifacts"
	@echo ""
	@echo "Running:"
	@echo "  make gateway    Start gateway server"
	@echo "  make cli        Start CLI"
	@echo "  make agent      Run agent"
	@echo ""
	@echo "Deployment:"
	@echo "  make docker     Build Docker image"
	@echo "  make deploy     Deploy to production"

install:
	npm install

setup:
	node scripts/setup/full-setup.js

build:
	npm run build

rebuild: clean build

dev:
	npm run dev

test:
	npm test

test-watch:
	npm run test:watch

coverage:
	npm test -- --coverage

lint:
	npm run lint

format:
	npm run format

clean:
	rm -rf dist/ node_modules/ .coverage/

gateway:
	npm run gateway

cli:
	npm start

agent:
	npm run agent

docker:
	docker build -t drogonclaw:latest .

deploy: build
	@echo "Deploying DrogonClaw..."
	@echo "Run 'npm run gateway' to start"

.DEFAULT_GOAL := help
