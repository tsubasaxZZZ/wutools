## Usage
```
  -c int
        Specific max downloadconcurrent num(default:10) (default 10)
  -f string
        (Not Implement)Specific CSV file
  -metadata-only
        If you want to get only metadata, specific this option
  -n string
        Specific KB NO(if you want to multiple, separate comma)
```

## Example
### Download KB
- Default concurrent(10)
```
 .\kbdownloader.exe -n 4163920,4093105,4103714
```
- Specific concurrent
```
 .\kbdownloader.exe -n 4163920,4093105,4103714 -c 20
```
### Download metadata
Output to "metadata.csv" file
```
 .\kbdownloader.exe -n 4163920,4093105,4103714 --metadata-only
```

## Specification
- If already exist file, skip download. So you may need delete before run this tool.


## ToDo
- Support filter function
- Telemetry by Application Insights
- KB no from CSV
- Web UI