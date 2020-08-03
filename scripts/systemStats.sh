#!/bin/sh

MEMORY_STATS=$(cat /proc/meminfo | sed 's/://g' | awk '{ print "\""$1"\":{\"value\":"$2",\"unit\":\""$3"\"},"}' | tr -d '\n' | sed 's/,$//')
IO_STATS=$(iostat -d -k | grep "[0-9]$" | sed 's/,/./g' | awk '{print "\""$1"\":{\"tps\":"$2",\"read/s\":"$3",\"wrtn/s\":"$4",\"read\":"$5",\"wrtn\":"$6"},"}' | tr -d '\n' | sed 's/,$//')
DISK_STATS=$(df -P | grep -v "^tmpfs" | grep -v "^shm" | grep -v "^run" | grep -v "^dev" | sed 1d | awk '{print "\""$6"\":{\"fs\":\""$1"\",\"used\":"$3",\"free\":"$4",\"total\":"$3+$4"},"}' | tr -d '\n' | sed 's/,$//')
NETWORK_STATS=$(ifstat -j | sed 's/^{//' | sed 's/}$//')
LOAD_AVG=$(uptime | grep -o "load average: .*$" | cut -d\: -f2 | sed 's/, / /g' | sed 's/,/./g' | awk '{print $1", "$2", "$3}')

TOP_OUT=$(echo "1" | top -n1 2> /dev/null | grep "^CPU" | sed 's/CPU//g' | sed 's/\%//g' | awk '{print "\""$1"\":{\"%usr\":"$2",\"%sys\":"$4",\"%nice\":"$6",\"%idle\":"$8",\"%iowait\":"$10",\"%irq\":"$12",\"%soft\":"$14"},"}' | tr -d '\n')

if [ "$TOP_OUT" != "" ]; then
  CPU_ALL=$(top -n1  | grep "^CPU" | sed 's/CPU//g' | sed 's/\%//g' | awk '{print "\"all\":{\"%usr\":"$2",\"%sys\":"$4",\"%nice\":"$6",\"%idle\":"$8",\"%iowait\":"$10",\"%irq\":"$12",\"%soft\":"$14"}"}'  | tr -d '\n' | sed 's/,$//')
  CPU_STATS=${TOP_OUT}${CPU_ALL}
else
  CPU_STATS=$(mpstat -P ALL | grep "^[0-9]" | grep -v "CPU" | sed 's/,/./g' | awk '{print "\""$2"\":{\"%usr\":"$3",\"%nice\":"$4",\"%sys\":"$5",\"%iowait\":"$6",\"%irq\":"$7",\"%soft\":"$8",\"%steal\":"$9",\"%guest\":"$10",\"%idle\":"$11"},"}' | tr -d '\n' | sed 's/,$//')
fi

read -d '' JSON << EOF
{
    "timestamp": $(date +%s),
    "mem": {
        ${MEMORY_STATS}
    },
    "cpu": {
        ${CPU_STATS}
    },
    "io": {
        ${IO_STATS}
    },
    "disk": {
        ${DISK_STATS}
    },
    "net": {
        ${NETWORK_STATS}
    },
    "load": [${LOAD_AVG}]
}
EOF

echo ${JSON} | jq -c
