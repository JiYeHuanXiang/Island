#!/bin/zsh
# 确保 parser.y 被正确解析
goyacc -o parser.go -v y.output parser.y

# 合并文件
cat parser.go >> parser.go.tmp
mv parser.go.tmp parser.go