<template>
  <div class="service-deploy page-shell page-shell--crud">
    <div class="page-header">
      <h2>服务部署</h2>
      <p>单页面完成服务部署与通用基础组件配置</p>
    </div>

    <el-card class="deploy-card">
      <el-form
        ref="deployFormRef"
        :model="deployForm"
        :rules="deployRules"
        label-position="top"
      >
        <el-row :gutter="16">
          <el-col :xs="24" :md="12">
            <el-form-item label="部署名称" prop="name">
              <el-input v-model="deployForm.name" placeholder="如：orders-api" clearable />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="服务类型">
              <el-select v-model="deployForm.type" style="width: 100%">
                <el-option label="Docker" value="docker" />
                <el-option label="Kubernetes" value="k8s" />
                <el-option label="Linux Service" value="linux" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :xs="24" :md="16">
            <el-form-item label="镜像地址" prop="image">
              <el-input v-model="deployForm.image" placeholder="registry.example.com/team/app:tag" clearable />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="8">
            <el-form-item label="服务端口" prop="port">
              <el-input-number v-model="deployForm.port" :min="1" :max="65535" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :xs="24" :md="8">
            <el-form-item label="副本数量" prop="replicas">
              <el-input-number v-model="deployForm.replicas" :min="1" :max="100" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="16">
            <el-form-item label="描述">
              <el-input v-model="deployForm.description" placeholder="服务用途、依赖说明（可选）" clearable />
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider>基础组件</el-divider>
        <el-form-item label="通用基础组件（可多选）">
          <el-checkbox-group v-model="deployForm.baseComponents">
            <el-checkbox v-for="item in baseComponentOptions" :key="item.value" :label="item.value">
              {{ item.label }}
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>

        <el-divider>资源与健康检查</el-divider>
        <el-row :gutter="16">
          <el-col :xs="24" :md="12">
            <el-form-item label="CPU Request (m)">
              <el-input-number v-model="deployForm.resources.cpuRequestMilli" :min="50" :step="50" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="CPU Limit (m)">
              <el-input-number v-model="deployForm.resources.cpuLimitMilli" :min="100" :step="100" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :xs="24" :md="12">
            <el-form-item label="Memory Request (Mi)">
              <el-input-number v-model="deployForm.resources.memoryRequestMi" :min="128" :step="64" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="Memory Limit (Mi)">
              <el-input-number v-model="deployForm.resources.memoryLimitMi" :min="256" :step="128" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :xs="24" :md="12">
            <el-form-item label="Readiness Path">
              <el-input v-model="deployForm.probe.readinessPath" placeholder="/healthz" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="Liveness Path">
              <el-input v-model="deployForm.probe.livenessPath" placeholder="/livez" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider>环境变量</el-divider>
        <el-form-item>
          <div class="env-vars">
            <div v-for="(env, index) in deployForm.envVars" :key="index" class="env-item">
              <el-input v-model="env.key" placeholder="变量名" class="env-key" />
              <el-input v-model="env.value" placeholder="变量值" class="env-value" />
              <el-button type="danger" size="small" @click="removeEnvVar(index)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
            <el-button type="primary" size="small" @click="addEnvVar">
              <el-icon><Plus /></el-icon>
              添加环境变量
            </el-button>
          </div>
        </el-form-item>

        <el-divider>数据卷</el-divider>
        <el-form-item>
          <div class="volumes">
            <div v-for="(volume, index) in deployForm.volumes" :key="index" class="volume-item">
              <el-input v-model="volume.name" placeholder="卷名" class="volume-name" />
              <el-input v-model="volume.mountPath" placeholder="挂载路径" class="volume-mount" />
              <el-input v-model="volume.hostPath" placeholder="主机路径" class="volume-host" />
              <el-button type="danger" size="small" @click="removeVolume(index)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
            <el-button type="primary" size="small" @click="addVolume">
              <el-icon><Plus /></el-icon>
              添加数据卷
            </el-button>
          </div>
        </el-form-item>

        <el-form-item class="submit-item">
          <el-button type="primary" :loading="loading" @click="handleDeploy">
            <el-icon><Upload /></el-icon>
            部署服务
          </el-button>
          <el-button @click="handleReset">
            <el-icon><RefreshRight /></el-icon>
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { Upload, Delete, Plus, RefreshRight } from '@element-plus/icons-vue'
import { useServiceStore } from '../../stores/service'
import type { DeployServiceParams } from '../../types/service'

const serviceStore = useServiceStore()

const baseComponentOptions = [
  { label: 'Nginx 网关', value: 'nginx' },
  { label: 'Redis 缓存', value: 'redis' },
  { label: 'MySQL 数据库', value: 'mysql' },
  { label: 'Kafka 消息队列', value: 'kafka' },
  { label: 'Prometheus 监控', value: 'prometheus' },
  { label: 'Node Exporter', value: 'node-exporter' }
]

const deployForm = reactive({
  name: '',
  type: 'docker' as 'docker' | 'k8s' | 'linux',
  description: '',
  image: '',
  replicas: 1,
  port: 80,
  baseComponents: [] as string[],
  resources: {
    cpuRequestMilli: 200,
    cpuLimitMilli: 1000,
    memoryRequestMi: 256,
    memoryLimitMi: 1024
  },
  probe: {
    readinessPath: '/healthz',
    livenessPath: '/livez'
  },
  envVars: [
    { key: '', value: '' }
  ],
  volumes: [
    { name: '', mountPath: '', hostPath: '' }
  ]
})

const deployRules = reactive({
  name: [
    { required: true, message: '请输入部署名称', trigger: 'blur' },
    { min: 2, max: 50, message: '部署名称长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  image: [
    { required: true, message: '请输入Docker镜像地址', trigger: 'blur' }
  ],
  replicas: [
    { required: true, message: '请输入副本数量', trigger: 'blur' },
    { type: 'number', min: 1, message: '副本数量至少为1', trigger: 'blur' }
  ],
  port: [
    { required: true, message: '请输入服务端口', trigger: 'blur' },
    { type: 'number', min: 1, max: 65535, message: '端口范围在 1 到 65535', trigger: 'blur' }
  ]
})

const deployFormRef = ref()
const loading = ref(false)

const addEnvVar = () => {
  deployForm.envVars.push({ key: '', value: '' })
}

const removeEnvVar = (index: number) => {
  deployForm.envVars.splice(index, 1)
  if (deployForm.envVars.length === 0) {
    deployForm.envVars.push({ key: '', value: '' })
  }
}

const addVolume = () => {
  deployForm.volumes.push({ name: '', mountPath: '', hostPath: '' })
}

const removeVolume = (index: number) => {
  deployForm.volumes.splice(index, 1)
  if (deployForm.volumes.length === 0) {
    deployForm.volumes.push({ name: '', mountPath: '', hostPath: '' })
  }
}

const handleDeploy = async () => {
  if (!deployFormRef.value) return

  try {
    await deployFormRef.value.validate()
    loading.value = true

    const env = deployForm.envVars.reduce((acc, item) => {
      if (item.key) {
        acc[item.key] = item.value
      }
      return acc
    }, {} as Record<string, string>)

    const volume = deployForm.volumes
      .filter(item => item.name && item.mountPath)
      .map(item => ({
        name: item.name,
        mountPath: item.mountPath,
        hostPath: item.hostPath
      }))

    const config = {
      components: deployForm.baseComponents,
      env,
      volume,
      resources: deployForm.resources,
      probe: deployForm.probe,
      service: {
        port: deployForm.port
      }
    }

    const deployData: DeployServiceParams = {
      name: deployForm.name,
      type: deployForm.type,
      description: deployForm.description,
      image: deployForm.image,
      replicas: deployForm.replicas,
      port: deployForm.port,
      config
    }

    const res = await serviceStore.deployNewService(deployData)
    if (res) {
      ElMessage.success('服务部署成功')
      handleReset()
    } else {
      ElMessage.error('服务部署失败')
    }

  } catch (error) {
    console.error('部署失败:', error)
    ElMessage.error('服务部署失败，请检查表单信息')
  } finally {
    loading.value = false
  }
}

const handleReset = () => {
  Object.assign(deployForm, {
    name: '',
    type: 'docker',
    description: '',
    image: '',
    replicas: 1,
    port: 80,
    baseComponents: [],
    resources: {
      cpuRequestMilli: 200,
      cpuLimitMilli: 1000,
      memoryRequestMi: 256,
      memoryLimitMi: 1024
    },
    probe: {
      readinessPath: '/healthz',
      livenessPath: '/livez'
    },
    envVars: [{ key: '', value: '' }],
    volumes: [{ name: '', mountPath: '', hostPath: '' }]
  })

  if (deployFormRef.value) {
    deployFormRef.value.resetFields()
  }
}
</script>

<style scoped>
.service-deploy {
  padding: 0;
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
  box-sizing: border-box;
  overflow-y: auto;
  overflow-x: hidden;
}

.page-header {
  margin-bottom: 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e5e7eb;
}

.page-header h2 {
  margin: 0;
  color: var(--el-color-primary);
  font-size: 20px;
  font-weight: 600;
}

.page-header p {
  margin: 6px 0 0;
  color: #6b7280;
}

.deploy-card {
  width: 100%;
  max-width: none;
  margin: 0;
  box-sizing: border-box;
  flex: 1;
  overflow-y: auto;
}

.env-vars, .volumes {
  display: flex;
  flex-direction: column;
  gap: 15px;
  width: 100%;
}

.env-item, .volume-item {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
  width: 100%;
  box-sizing: border-box;
}

.env-key, .volume-name {
  flex: 1;
  min-width: 180px;
  max-width: 250px;
}

.env-value, .volume-mount, .volume-host {
  flex: 1;
  min-width: 220px;
  max-width: 350px;
}

.submit-item {
  margin-top: 20px;
  text-align: center;
  width: 100%;
}
</style>
