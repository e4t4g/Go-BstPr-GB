report:
	go test -v 2>&1 |go-junit-report > report.xml

lint:
	 golangci-lint run -c golangci-lint.yaml

