set dotenv-load := true
set dotenv-path := "{{justfile_directory()}}/.env"

# default recipe to show all recipes
default:
	@just --list

[group('codegen')]
codegen:
	echo "codegen is disabled"
	# cd {{justfile_directory()}}/cmd/py_faas && go tool oapi-codegen -config oapi-codegen-config.yaml oapi-spec.yaml


# build the named services
[group('build')]
build binary: codegen
	go build -o {{justfile_directory()}}/bin/"{{binary}}" {{justfile_directory()}}/cmd/"{{binary}}"
	chmod +x {{justfile_directory()}}/bin/"{{binary}}"

# keep the code tidy
[group('build')]
[group('test')]
lint:
	go mod tidy
	golangci-lint run --fix
	golangci-lint run

# run locally
[group('test')]
dev binary: (build binary)
	{{justfile_directory()}}/bin/"{{binary}}"