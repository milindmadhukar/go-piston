
go clean -testcache && go test ./... -v -cover | sed ''/PASS/s//$(printf "\033[32mPASS\033[0m")/'' | sed ''/FAIL/s//$(printf "\033[31mFAIL\033[0m")/'' | sed ''/RUN/s//$(printf "\033[33mRUN\033[0m")/'' | GREP_COLOR='01;32' grep -E --color 'ok.*$|^.*ok.*$|$'
