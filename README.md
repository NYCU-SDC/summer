# summer
## Get Started
Summer provide a CLI tool to help you initialize the project.
### First, Install the CLI tool:
```
go install github.com/NYCU-SDC/summer/cmd/summer@latest
```
To verify the installation, run:
```
summer -v
```

### Initialize the project
```
summer -b main init
```
You may choose any other branch with -b flag.

Then, summer will ask for the project name. The project name will be used as the module name in go.mod. You can still edit it.

Summer will create a file structure like bellow:
```
.
├── cmd/
│   └── main.go
├── internal
└── scripts/
    └── create_full_schema.sh
```
`main.go` will containe a minimal server example.
`create_full_schema.sh` is a recommended helper script if you would like to use `sqlc` for SQL generation. The usage will be described in later section.

### Run the example project
First install the dependencies:
```
go mod tidy
```
Then start the server:
```
go run ./cmd/main.go
```
Finally try it out. Use any api testing tool and hit the following endpoint:
```
localhost:8080/healthz
```
You will recive a greeting from the local server!

### Use the shell script
The `create_full_schema.sh` is use to collect all `schema.sql` in `internal` folder and output, merge them all to `./internal/database/full_schema.sql`.

This is a recommeded tool if you want to use sqlc for SQL generation. Edit the shell script if you wish to use different output file or change other behaviors.
