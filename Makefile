budget-reporter.so budget-reporter.h: budget-reporter.go
	GOOS=darwin GOARCH=arm64 go build -o budget-reporter.so -buildmode=c-shared budget-reporter.go
	pipenv run python3 test.py

clean:
	rm -f budget-reporter.h budget-reporter.so