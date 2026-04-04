# ── 编译阶段 ──────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# 1. 运行 Excel 转换工具，生成 characters.json 和 fields.json
# 如果这一步报错，说明 Excel 路径或格式有问题，构建会直接停止，防止部署错误代码
RUN go run ./cmd/exceltojson/main.go -i data/游戏配置表.xlsx -o data/

# 2. 编译主程序
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o server .

# ── 运行阶段 ──────────────────────────────────────────────────
FROM alpine:3.20

WORKDIR /app

# 3. 拷贝主程序
COPY --from=builder /app/server .

# 4. 拷贝转换生成的 JSON 数据文件（保持目录结构）
# 这样程序启动时执行 character.LoadFromFile("./data/characters.json") 就能找到了
COPY --from=builder /app/data/*.json ./data/

EXPOSE 43966
ENTRYPOINT ["./server"]
