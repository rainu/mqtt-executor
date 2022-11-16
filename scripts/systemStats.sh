#!/bin/sh

PRINT_CPU=0
PRINT_MEMORY=0
PRINT_IO=0
PRINT_DISK=0
PRINT_NETWORK=0
PRINT_LOAD=0

while getopts 'acmidnl' OPTION; do
  case $OPTION in
    a)
      PRINT_CPU=1
      PRINT_MEMORY=1
      PRINT_IO=1
      PRINT_DISK=1
      PRINT_NETWORK=1
      PRINT_LOAD=1
    ;;
    c)
    PRINT_CPU=1
    ;;
    m)
    PRINT_MEMORY=1
    ;;
    i)
    PRINT_IO=1
    ;;
    d)
    PRINT_DISK=1
    ;;
    n)
    PRINT_NETWORK=1
    ;;
    l)
    PRINT_LOAD=1
    ;;
  esac
done

if [ $PRINT_MEMORY -eq 1 ]; then
  MEMORY_STATS=$(free | tail -2 | head -1 | awk '{print "\"total\":"$2",\"used\":"$3",\"free\":"$4",\"shared\":"$5",\"buff/cache\":"$6",\"available\":"$7""}')
fi

if [ $PRINT_IO -eq 1 ]; then
  IO_STATS=$(iostat -d -k | grep "[0-9]$" | sed 's/,/./g' | awk '{print "\""$1"\":{\"tps\":"$2",\"read/s\":"$3",\"wrtn/s\":"$4",\"read\":"$5",\"wrtn\":"$6"},"}' | tr -d '\n' | sed 's/,$//')
fi

if [ $PRINT_DISK -eq 1 ]; then
  DISK_STATS=$(df -P | grep -v "^tmpfs" | grep -v "^shm" | grep -v "^run" | grep -v "^dev" | sed 1d | awk '{print "\""$6"\":{\"fs\":\""$1"\",\"used\":"$3",\"free\":"$4",\"total\":"$3+$4"},"}' | tr -d '\n' | sed 's/,$//')
fi

if [ $PRINT_NETWORK -eq 1 ]; then
  NETWORK_STATS=$(ifstat -j | sed 's/^{//' | sed 's/}$//')
fi

if [ $PRINT_LOAD -eq 1 ]; then
  LOAD_AVG=$(uptime | grep -o "load average: .*$" | cut -d\: -f2 | sed 's/, / /g' | sed 's/,/./g' | awk '{print $1", "$2", "$3}')
fi

if [ $PRINT_CPU -eq 1 ]; then
  TOP_OUT=$(echo "1" | top -n1 2> /dev/null | grep "^CPU" | sed 's/CPU//g' | sed 's/\%//g' | awk '{print "\""$1"\":{\"%usr\":"$2",\"%sys\":"$4",\"%nice\":"$6",\"%idle\":"$8",\"%iowait\":"$10",\"%irq\":"$12",\"%soft\":"$14"},"}' | tr -d '\n')

  if [ "$TOP_OUT" != "" ]; then
    CPU_ALL=$(top -n1  | grep "^CPU" | sed 's/CPU//g' | sed 's/\%//g' | awk '{print "\"all\":{\"%usr\":"$2",\"%sys\":"$4",\"%nice\":"$6",\"%idle\":"$8",\"%iowait\":"$10",\"%irq\":"$12",\"%soft\":"$14"}"}'  | tr -d '\n' | sed 's/,$//')
    CPU_STATS=${TOP_OUT}${CPU_ALL}
  else
    CPU_STATS=$(mpstat -P ALL | grep "^[0-9]" | grep -v "CPU" | sed 's/,/./g' | awk '{print "\""$2"\":{\"%usr\":"$3",\"%nice\":"$4",\"%sys\":"$5",\"%iowait\":"$6",\"%irq\":"$7",\"%soft\":"$8",\"%steal\":"$9",\"%guest\":"$10",\"%idle\":"$11"},"}' | tr -d '\n' | sed 's/,$//')
  fi
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

echo ${JSON} | jq 'walk(if type == "object" then with_entries(select(.value | (. != {} and . != []))) else . end)' -c
