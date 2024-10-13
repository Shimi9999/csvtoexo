module github.com/Shimi9999/csvtoexo

go 1.17

exclude (
	// include vulnerability CVE-2022-32149
	golang.org/x/text v0.3.0
	golang.org/x/text v0.3.3
	golang.org/x/text v0.3.7
)

require golang.org/x/text v0.19.0
