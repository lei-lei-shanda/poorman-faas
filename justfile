set dotenv-load := true
set dotenv-path := "{{justfile_directory()}}/.env"

# default recipe to show all recipes
default:
	@just --list

# build the named services
[group('build')]
build binary:
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