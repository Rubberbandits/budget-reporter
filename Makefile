budget-reporter: budget-reporter.go
	GOOS=darwin GOARCH=arm64 go build budget-reporter.go
	./budget-reporter

clean:
	rm -f budget-reporter