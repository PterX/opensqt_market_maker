# Release Guide

## 版本规范

- 版本号以 [main.go](main.go#L21) 中的 `Version` 为唯一来源，格式固定为 `v主版本.次版本.修订号`，例如 `v3.4.2`。
- Git 标签必须与 `Version` 完全一致。
- GitHub Release 名称、Tag 名称、发行包文件名必须与该版本号保持一致。
- [ARCHITECTURE.md](ARCHITECTURE.md) 顶部文档版本建议同步更新，避免文档和程序版本脱节。
- 如果某个标签已经发布，后续改动必须递增补丁版本，不要重打已有标签。

## 发布流程

1. 修改 [main.go](main.go#L21) 中的版本号。
2. 如有必要，同步更新 [ARCHITECTURE.md](ARCHITECTURE.md) 中的版本说明。
3. 验证模块和编译：

```bash
go mod verify
go build ./...
```

4. 提交代码并推送到主分支。
5. 创建并推送同名标签，例如：

```bash
git tag -a v3.4.2 -m "Release v3.4.2"
git push origin main
git push origin v3.4.2
```

6. 生成发行包：

```bash
./scripts/package_release.sh
```

7. 将 `dist/` 下生成的 `.tar.gz` 和 `.sha256` 上传到 GitHub Release。

## 发行包内容

脚本 [scripts/package_release.sh](scripts/package_release.sh) 会生成以下内容：

- 编译后的可执行文件 `opensqt_market_maker`
- 整个 `live_server/` 目录
- `config.example.yaml`
- `config.yaml`
- `README.md`
- `ARCHITECTURE.md`
- `部署教程.pdf`

## 安全说明

- 发布包中的 `config.yaml` 由 `config.example.yaml` 复制生成，不会使用你本机的真实 [config.yaml](config.yaml)。
- 这样可以保留开箱即用的目录结构，同时避免把 API Key 或私钥打进公开 Release。

## 推荐命名

- 发行包：`opensqt_market_maker_v3.4.2_linux_amd64.tar.gz`
- 校验文件：`opensqt_market_maker_v3.4.2_linux_amd64.tar.gz.sha256`
