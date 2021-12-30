#!/bin/bash
echo "xxl-job: hello shell"

echo "脚本位置：$0"
echo "任务参数：$1"
echo "分片序号 = $2"
echo "分片总数 = $3"

for i in {1..5} ; do
  echo "sleep 3"
  sleep 3
done

echo "Good bye!"
exit 0
