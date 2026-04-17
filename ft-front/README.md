# ft-front

一个现代化的服务器管理平台前端项目，基于 Vue 3 + TypeScript + Element Plus 构建，提供机器管理、Kubernetes 部署、监控配置等功能。

## 技术栈

- **框架**: Vue 3 + Composition API
- **语言**: TypeScript
- **UI 组件库**: Element Plus
- **构建工具**: Vite
- **状态管理**: Pinia
- **路由管理**: Vue Router 4
- **HTTP 客户端**: Axios
- **进度条**: NProgress

## 功能特性

### 核心功能
- **仪表盘**: 系统概览和关键指标展示
- **机器管理**: 服务器资源监控和管理
- **Kubernetes 部署**: 集群创建和节点配置
- **服务管理**: Linux 服务部署和管理
- **监控配置**: Prometheus 和各种 Exporter 配置

### 高级功能
- **安全审计**: 操作日志和权限管理
- **初始化工具**: 系统参数优化、磁盘分区、安全加固等
- **备份恢复**: 数据备份和恢复功能
- **性能分析**: 系统性能监控和分析

## 快速开始

### 环境要求
- Node.js >= 18.0.0
- npm >= 9.0.0

### 安装依赖
```bash
npm install
```

### 开发模式
```bash
npm run dev
```

项目将在 `http://localhost:5173` 启动。

### 构建生产版本
```bash
npm run build
```

构建后的文件将输出到 `dist` 目录。

### 预览生产版本
```bash
npm run preview
```

### 开发模式监听文件变化
```bash
npm run watch
```

## 项目结构

```
ft-front/
├── public/                 # 静态资源
├── src/
│   ├── api/               # API 接口定义
│   ├── assets/            # 项目资源文件
│   ├── components/        # 公共组件
│   │   ├── common/        # 通用组件
│   │   ├── k8s/           # Kubernetes 相关组件
│   │   └── layout/        # 布局组件
│   ├── mock/              # Mock 数据
│   ├── router/            # 路由配置
│   ├── stores/            # Pinia 状态管理
│   ├── types/             # TypeScript 类型定义
│   ├── utils/             # 工具函数
│   ├── views/             # 页面组件
│   │   ├── advanced/      # 高级功能
│   │   ├── init-tools/    # 初始化工具
│   │   ├── job/           # 任务中心
│   │   ├── login/         # 登录页
│   │   ├── machine/       # 机器管理
│   │   ├── monitoring/    # 监控配置
│   │   ├── proxy/         # 代理配置
│   │   ├── security-audit/# 安全审计
│   │   ├── service/       # 服务管理
│   │   │   └── k8s-deploy/# Kubernetes 部署
│   │   └── user/          # 用户管理
│   ├── App.vue            # 根组件
│   ├── main.ts            # 入口文件
│   └── style.css          # 全局样式
├── .env                   # 环境变量配置
├── .gitignore             # Git 忽略文件
├── index.html             # HTML 入口
├── package.json           # 项目配置
├── tsconfig.json          # TypeScript 配置
├── vite.config.ts         # Vite 配置
└── README.md              # 项目说明
```

## 配置说明

### 环境变量
在 `.env` 文件中配置环境变量：

```env
# API 基础 URL
VITE_API_BASE_URL=http://localhost:8000/api

# 其他配置...
```

### API 接口
所有 API 接口定义在 `src/api/` 目录下，使用 Axios 进行 HTTP 请求。

### 路由配置
路由配置在 `src/router/index.ts` 中，使用 Vue Router 4 进行路由管理。

### 状态管理
使用 Pinia 进行状态管理，所有状态定义在 `src/stores/` 目录下。

## 开发规范

### 代码规范
- 使用 TypeScript 编写代码
- 遵循 Vue 3 Composition API 风格
- 组件命名使用 PascalCase
- 文件命名使用 kebab-case

### 提交规范
- 使用 Conventional Commits 规范
- 提交信息格式：`type(scope): description`

## 浏览器支持

- Chrome (最新版本)
- Firefox (最新版本)
- Safari (最新版本)
- Edge (最新版本)

## 许可证

MIT License
