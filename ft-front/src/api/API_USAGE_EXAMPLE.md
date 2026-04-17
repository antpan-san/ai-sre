# API接口管理使用示例

## 概述

本项目已将所有API接口整合到统一的`src/api/index.ts`文件中，实现了接口信息与接口实现的分离，便于统一管理和维护。

## API接口标准信息

每个API接口都定义了标准信息，包括：

- `name`: 接口名称
- `method`: 请求方法（GET/POST/PUT/DELETE/PATCH）
- `url`: 请求地址
- `description`: 接口描述
- `requestType`: 请求参数类型
- `responseType`: 响应参数类型
- `module`: 所属模块

## 接口分类

所有接口按照功能模块进行分类：

- **API_AUTH**: 认证相关接口
- **API_DASHBOARD**: 仪表盘相关接口
- **API_MACHINE**: 机器管理相关接口
- **API_SERVICE**: 服务管理相关接口
- **API_K8S_DEPLOY**: Kubernetes部署相关接口
- **API_MONITORING**: 监控告警相关接口
- **API_PROXY**: 代理配置相关接口
- **API_USER**: 用户管理相关接口

## 使用方式

### 1. 导入API接口

```typescript
// 导入所有API接口和接口信息
import { 
  // API请求函数
  login, getUserInfo, getMachineList, 
  // API接口标准信息
  API_AUTH, API_MACHINE 
} from '@/api/index'
```

### 2. 使用API请求函数

```typescript
// 用户登录
const handleLogin = async (form: LoginForm) => {
  try {
    const response = await login(form)
    // 处理登录成功逻辑
  } catch (error) {
    // 处理错误
  }
}

// 获取机器列表
const fetchMachines = async () => {
  try {
    const params: MachineListParams = {
      page: 1,
      pageSize: 10
    }
    const response = await getMachineList(params)
    // 处理机器列表数据
  } catch (error) {
    // 处理错误
  }
}
```

### 3. 查看API接口标准信息

```typescript
// 查看登录接口的标准信息
console.log('登录接口信息:', API_AUTH.login)
// 输出:
// {
//   name: 'login',
//   method: 'POST',
//   url: '/api/auth/login',
//   description: '用户登录，获取访问令牌',
//   requestType: 'LoginForm',
//   responseType: 'LoginResponse',
//   module: '认证'
// }

// 查看机器列表接口的标准信息
console.log('机器列表接口信息:', API_MACHINE.getMachineList)
// 输出:
// {
//   name: 'getMachineList',
//   method: 'GET',
//   url: '/api/machine',
//   description: '获取所有机器的列表数据，支持分页和筛选',
//   requestType: 'MachineListParams',
//   responseType: 'MachineListResponse',
//   module: '机器管理'
// }
```

### 4. JSON格式接口信息

每个接口的标准信息可以序列化为JSON格式，便于存储和传输：

```json
// 登录接口的JSON格式信息
{
  "name": "login",
  "method": "POST",
  "url": "/api/auth/login",
  "description": "用户登录，获取访问令牌",
  "requestType": "LoginForm",
  "responseType": "LoginResponse",
  "module": "认证"
}

// 机器列表接口的JSON格式信息
{
  "name": "getMachineList",
  "method": "GET",
  "url": "/api/machine",
  "description": "获取所有机器的列表数据，支持分页和筛选",
  "requestType": "MachineListParams",
  "responseType": "MachineListResponse",
  "module": "机器管理"
}

// 认证模块所有接口的JSON格式信息
{
  "login": {
    "name": "login",
    "method": "POST",
    "url": "/api/auth/login",
    "description": "用户登录，获取访问令牌",
    "requestType": "LoginForm",
    "responseType": "LoginResponse",
    "module": "认证"
  },
  "logout": {
    "name": "logout",
    "method": "POST",
    "url": "/api/auth/logout",
    "description": "用户登出，清除会话信息",
    "requestType": "void",
    "responseType": "void",
    "module": "认证"
  },
  "getUserInfo": {
    "name": "getUserInfo",
    "method": "GET",
    "url": "/api/auth/info",
    "description": "获取当前登录用户的详细信息",
    "requestType": "void",
    "responseType": "User",
    "module": "认证"
  }
}
```

### 5. 批量获取模块接口信息

```typescript
// 获取认证模块所有接口信息
console.log('认证模块所有接口:', API_AUTH)

// 获取机器管理模块所有接口信息
console.log('机器管理模块所有接口:', API_MACHINE)
```

## 接口列表

### 认证相关接口

| 接口名称 | 请求方法 | 请求地址 | 接口描述 |
|---------|---------|---------|---------|
| login | POST | /api/auth/login | 用户登录，获取访问令牌 |
| logout | POST | /api/auth/logout | 用户登出，清除会话信息 |
| getUserInfo | GET | /api/auth/info | 获取当前登录用户的详细信息 |

### 机器管理相关接口

| 接口名称 | 请求方法 | 请求地址 | 接口描述 |
|---------|---------|---------|---------|
| getMachineList | GET | /api/machine | 获取所有机器的列表数据，支持分页和筛选 |
| getMachineDetail | GET | /api/machine/:id | 获取指定机器的详细信息 |
| addMachine | POST | /api/machine | 添加新的机器信息 |
| updateMachine | PUT | /api/machine/:id | 更新指定机器的信息 |
| deleteMachine | DELETE | /api/machine/:id | 删除指定的机器信息 |
| batchDeleteMachine | DELETE | /api/machine/batch | 批量删除多个机器信息 |
| updateMachineStatus | PATCH | /api/machine/:id/status | 更新指定机器的在线状态 |

### 服务管理相关接口

| 接口名称 | 请求方法 | 请求地址 | 接口描述 |
|---------|---------|---------|---------|
| deployService | POST | /api/service/deploy | 部署新的服务 |
| getServiceList | GET | /api/service/list | 获取所有服务的列表数据，支持分页和筛选 |
| getServiceDetail | GET | /api/service/detail | 获取指定服务的详细信息 |
| startService | POST | /api/service/start | 启动指定的服务 |
| stopService | POST | /api/service/stop | 停止指定的服务 |
| restartService | POST | /api/service/restart | 重启指定的服务 |
| deleteService | DELETE | /api/service/delete | 删除指定的服务 |
| batchDeleteServices | POST | /api/service/batch-delete | 批量删除多个服务 |
| getLinuxServiceList | GET | /api/service/linux/list | 获取Linux系统服务列表 |
| operateLinuxService | POST | /api/service/linux/operate | 对Linux系统服务进行操作（启动/停止/重启/启用/禁用） |

### Kubernetes部署相关接口

| 接口名称 | 请求方法 | 请求地址 | 接口描述 |
|---------|---------|---------|---------|
| getK8sVersions | GET | /api/k8s/deploy/versions | 获取支持的Kubernetes版本列表 |
| getMachines | GET | /api/k8s/deploy/machines | 获取可用于部署K8s集群的机器列表 |
| checkClusterName | GET | /api/k8s/deploy/check-name | 校验集群名称是否唯一 |
| submitDeployConfig | POST | /api/k8s/deploy/submit | 提交Kubernetes集群部署配置 |
| getDeployProgress | GET | /api/k8s/deploy/progress | 获取Kubernetes集群部署进度 |
| getDeployLogs | GET | /api/k8s/deploy/logs | 获取Kubernetes集群部署日志 |

### 监控告警相关接口

| 接口名称 | 请求方法 | 请求地址 | 接口描述 |
|---------|---------|---------|---------|
| getMonitoringConfigList | GET | /api/monitoring/configs | 获取所有监控配置的列表 |
| getMonitoringConfig | GET | /api/monitoring/configs/:id | 获取指定监控配置的详细信息 |
| createMonitoringConfig | POST | /api/monitoring/configs | 创建新的监控配置 |
| updateMonitoringConfig | PUT | /api/monitoring/configs/:id | 更新指定的监控配置 |
| deleteMonitoringConfig | DELETE | /api/monitoring/configs/:id | 删除指定的监控配置 |
| getAlertRules | GET | /api/monitoring/alert-rules | 获取所有告警规则的列表 |
| createAlertRule | POST | /api/monitoring/alert-rules | 创建新的告警规则 |
| updateAlertRule | PUT | /api/monitoring/alert-rules/:id | 更新指定的告警规则 |
| deleteAlertRule | DELETE | /api/monitoring/alert-rules/:id | 删除指定的告警规则 |

### 代理配置相关接口

| 接口名称 | 请求方法 | 请求地址 | 接口描述 |
|---------|---------|---------|---------|
| getProxyConfigList | GET | /api/proxy/config/list | 获取所有代理配置的列表，支持分页和筛选 |
| getProxyConfigDetail | GET | /api/proxy/config/detail | 获取指定代理配置的详细信息 |
| saveProxyConfig | POST | /api/proxy/config/save | 保存代理配置（新增或更新） |
| deleteProxyConfig | DELETE | /api/proxy/config/delete | 删除指定的代理配置 |
| applyProxyConfig | POST | /api/proxy/config/apply | 应用指定的代理配置 |

### 用户管理相关接口

| 接口名称 | 请求方法 | 请求地址 | 接口描述 |
|---------|---------|---------|---------|
| getUserList | GET | /api/user | 获取所有用户的列表数据，支持分页和筛选 |
| getUserDetail | GET | /api/user/:id | 获取指定用户的详细信息 |
| addUser | POST | /api/user | 添加新的用户信息 |
| updateUser | PUT | /api/user/:id | 更新指定用户的信息 |
| deleteUser | DELETE | /api/user/:id | 删除指定的用户信息 |
| batchDeleteUser | DELETE | /api/user/batch | 批量删除多个用户信息 |
| updateUserRole | PATCH | /api/user/:id/role | 更新指定用户的角色 |

## 扩展新接口

### 1. 添加接口标准信息

在对应模块下添加新的接口标准信息：

```typescript
export const API_MACHINE = {
  // 已有接口...
  
  // 新接口
  newMachineApi: {
    name: 'newMachineApi',
    method: 'POST',
    url: '/api/machine/new',
    description: '新的机器接口',
    requestType: 'NewMachineParams',
    responseType: 'NewMachineResponse',
    module: '机器管理'
  }
}
```

### 2. 实现接口请求函数

在文件底部添加接口实现：

```typescript
export const newMachineApi = (data: NewMachineParams): Promise<NewMachineResponse> => {
  return request.post(API_MACHINE.newMachineApi.url, data)
}
```

## 注意事项

1. 所有API请求都经过了统一的错误处理和拦截器处理
2. 接口URL使用变量定义，便于统一修改和维护
3. 接口请求参数和响应参数都有明确的类型定义
4. 使用接口标准信息可以方便地生成API文档或进行接口测试
