module github.com/Shimi9999/csvtoexo

go 1.17

replace github.com/Shimi9999/csvtoexo/aviutlobj => ./aviutlobj

exclude (
	// include vulnerability CVE-2022-32149
	golang.org/x/text v0.3.0
	golang.org/x/text v0.3.3
	golang.org/x/text v0.3.7
)

require (
	github.com/Shimi9999/csvtoexo/aviutlobj v0.0.0-20230321062810-d6b54e36764c
	golang.org/x/text v0.19.0
)
