<template>
  <div class="service-deploy">
    <div class="page-header">
      <h2>服务部署</h2>
    </div>
    
    <el-card class="deploy-card">
      <el-form
        ref="deployFormRef"
        :model="deployForm"
        :rules="deployRules"
        label-position="top"
      >
        <el-form-item label="部署名称" prop="name">
          <el-input
            v-model="deployForm.name"
            placeholder="请输入部署名称"
            clearable
          />
        </el-form-item>
        
        <el-form-item label="镜像地址" prop="image">
          <el-input
            v-model="deployForm.image"
            placeholder="请输入Docker镜像地址"
            clearable
          />
        </el-form-item>
        
        <el-form-item label="副本数量" prop="replicas">
          <el-input-number
            v-model="deployForm.replicas"
            :min="1"
            :max="100"
            placeholder="请输入副本数量"
          />
        </el-form-item>
        
        <el-form-item label="端口" prop="port">
          <el-input-number
            v-model="deployForm.port"
            :min="1"
            :max="65535"
            placeholder="请输入服务端口"
          />
        </el-form-item>
        
        <el-form-item label="环境变量">
          <div class="env-vars">
            <div
              v-for="(env, index) in deployForm.envVars"
              :key="index"
              class="env-item"
            >
              <el-input
                v-model="env.key"
                placeholder="变量名"
                class="env-key"
              />
              <el-input
                v-model="env.value"
                placeholder="变量值"
                class="env-value"
              />
              <el-button
                type="danger"
                size="small"
                @click="removeEnvVar(index)"
              >
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
            
            <el-button type="primary" size="small" @click="addEnvVar">
              <el-icon><Plus /></el-icon>
              添加环境变量
            </el-button>
          </div>
        </el-form-item>
        
        <el-form-item label="数据卷">
          <div class="volumes">
            <div
              v-for="(volume, index) in deployForm.volumes"
              :key="index"
              class="volume-item"
            >
              <el-input
                v-model="volume.name"
                placeholder="卷名"
                class="volume-name"
              />
              <el-input
                v-model="volume.mountPath"
                placeholder="挂载路径"
                class="volume-mount"
              />
              <el-input
                v-model="volume.hostPath"
                placeholder="主机路径"
                class="volume-host"
              />
              <el-button
                type="danger"
                size="small"
                @click="removeVolume(index)"
              >
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
          <el-button
            type="primary"
            :loading="loading"
            @click="handleDeploy"
          >
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

// 服务管理Store
const serviceStore = useServiceStore()

// 部署表单
const deployForm = reactive({
  name: '',
  image: '',
  replicas: 1,
  port: 80,
  envVars: [
    { key: '', value: '' }
  ],
  volumes: [
    { name: '', mountPath: '', hostPath: '' }
  ]
})

// 部署表单验证规则
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

// 添加环境变量
const addEnvVar = () => {
  deployForm.envVars.push({ key: '', value: '' })
}

// 移除环境变量
const removeEnvVar = (index: number) => {
  deployForm.envVars.splice(index, 1)
  if (deployForm.envVars.length === 0) {
    deployForm.envVars.push({ key: '', value: '' })
  }
}

// 添加数据卷
const addVolume = () => {
  deployForm.volumes.push({ name: '', mountPath: '', hostPath: '' })
}

// 移除数据卷
const removeVolume = (index: number) => {
  deployForm.volumes.splice(index, 1)
  if (deployForm.volumes.length === 0) {
    deployForm.volumes.push({ name: '', mountPath: '', hostPath: '' })
  }
}

// 处理部署
const handleDeploy = async () => {
  if (!deployFormRef.value) return
  
  try {
    await deployFormRef.value.validate()
    loading.value = true
    
    // 转换环境变量格式
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
    
    // 准备部署数据
    const deployData: DeployServiceParams = {
      name: deployForm.name,
      image: deployForm.image,
      replicas: deployForm.replicas,
      port: deployForm.port,
      env,
      volume
    }
    
    // 调用部署API
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

// 处理重置
const handleReset = () => {
  Object.assign(deployForm, {
    name: '',
    image: '',
    replicas: 1,
    port: 80,
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
  padding: 16px;
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
  color: #1890ff;
  font-size: 20px;
  font-weight: 600;
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
