# new-api 安卓管理 App

原生安卓管理端（Kotlin + Jetpack Compose，Android 8.0 及以上），只使用服务端已有的管理接口。

- **使用记录**：按时间、类型、用户、令牌、模型、渠道、分组、请求 ID 筛选，查看计费明细。
- **用户**：搜索筛选，调整余额，改资料、分组和密码，启用/禁用，提升/降级，重置两步验证和 Passkey，删除，新建。
- **渠道**：搜索筛选，启用/禁用，测试，更新余额，新建和编辑。
- **我的**：账号和服务器信息，修改自己的密码，退出登录，检查更新（可选接收测试版）。

只有管理员和超级管理员能登录。站点开启了 Turnstile 或关闭了密码登录时，请在网页端头像菜单打开「个人资料」，在「安全 → 访问令牌」生成令牌，用令牌登录。

## 目录

- `core/`：纯 Kotlin 模块，包含接口模型、HTTP 客户端、登录与续期、额度格式化，以及单元测试和接口契约测试。
- `app/`：Compose 界面。

## 构建

需要 JDK 17 和 Android SDK（compileSdk 35）。

```bash
cd android
./gradlew :core:test            # 单元测试
./gradlew :app:assembleRelease  # 输出 app/build/outputs/apk/release/app-release.apk
```

契约测试会连到一个真实的服务端，没设置环境变量时跳过：

```bash
NEWAPI_IT_BASE_URL=http://127.0.0.1:3000 NEWAPI_IT_USERNAME=root NEWAPI_IT_PASSWORD=... ./gradlew :core:test
```

测试会创建并删除临时用户和渠道，请只对测试用的服务端运行。

## CI 与发布

`.github/workflows/android.yml` 在 PR 改到 `android/` 时运行，改动合并进 `main` 后也会运行：

- 用当前代码编译服务端，以 SQLite 启动，跑单元测试和契约测试。
- 编译 release APK，上传为 Actions artifact（`new-api-admin-apk`）。
- 测试通过且配置了签名密钥时，把 APK 发布到本仓库的 GitHub Releases：
  - PR 发布**测试版**：tag 为 `android-v<版本>-beta`，标记为 pre-release。PR 每推送一次就发布一个新的测试版。
  - 合并进 `main` 发布**正式版**：tag 为 `android-v<版本>`。
  - 两种都不会标记为 Latest，仓库的 Latest 仍然是服务端的版本。

### 版本号

`android/VERSION` 写主版本号和次版本号（如 `0.2`），第三段是这个工作流的运行序号，也是 APK 的 versionCode。测试版和正式版共用这个序号，所以新构建的版本号总比旧的大，可以直接覆盖安装。要升主版本或次版本时改 `android/VERSION`。

不要手动推送 `android-v` 开头的 tag：服务端的发布工作流会被 tag 推送触发。工作流用 `GITHUB_TOKEN` 创建的 tag 不会触发其他工作流，服务端工作流里的 `git describe` 也排除了 `android-v*`，避免版本号取到安卓的 tag。

### 应用内更新

App 会到本仓库的 GitHub Releases 检查更新：启动时自动检查（最多每 6 小时一次），也可以在「我的 → 检查更新」手动检查。默认只检查正式版，打开「接收测试版」后也会检查测试版。发现新版本时，App 下载 APK，校验大小和 SHA-256，然后交给系统安装。

- 检查更新调用不登录的 GitHub API，每个 IP 每小时限 60 次。
- 只有用签名密钥打包的 CI 构建才会检查更新。本地构建和没有配置签名密钥的构建签名不同，无法覆盖安装，所以不显示更新功能。
- 检查的仓库默认是构建时的 `GITHUB_REPOSITORY`，可以用环境变量 `ANDROID_UPDATE_REPOSITORY`（`owner/name`）覆盖。

### 签名密钥

签名密钥放在仓库的 Actions secrets（Repository secrets）里，四个都要配置：

| Secret | 内容 |
|---|---|
| `ANDROID_KEYSTORE_BASE64` | keystore 文件的 base64 |
| `ANDROID_KEYSTORE_PASSWORD` | keystore 密码 |
| `ANDROID_KEY_ALIAS` | 密钥别名 |
| `ANDROID_KEY_PASSWORD` | 密钥密码（PKCS12 格式的 keystore 与 keystore 密码相同） |

生成一次后长期使用，换了密钥的 APK 不能覆盖安装旧版本：

```bash
keytool -genkeypair -v -keystore release.jks -alias newapi-admin -keyalg RSA -keysize 2048 -validity 10000
base64 -w0 release.jks   # 输出填到 ANDROID_KEYSTORE_BASE64
```

没有配置时，APK 用每次构建临时生成的调试密钥签名，可以安装试用，但不能覆盖升级，也不会发布到 Releases。来自 fork 的 PR 拿不到这些密钥，同样只编译不发布。
