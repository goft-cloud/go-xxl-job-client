#!/bin/bash
echo "xxl-job: hello shell"

echo "全部参数 $*"
echo "脚本位置：$0"
echo "任务参数：$1"
echo "分片序号 = $2"
echo "分片总数 = $3"

echo "Good bye!"
exit 0
