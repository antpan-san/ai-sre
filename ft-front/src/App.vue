<script setup lang="ts">
import { RouterView, useRouter } from 'vue-router'
import NProgress from 'nprogress'
import 'nprogress/nprogress.css'

const router = useRouter()

// 配置 NProgress
NProgress.configure({
  showSpinner: false, // 隐藏加载图标
  trickleSpeed: 50,   // 进度条增长速度
  speed: 300          // 进度条显示/隐藏速度
})

// 路由开始前显示进度条
router.beforeEach((_to, _from, next) => {
  NProgress.start()
  next()
})

// 路由完成后隐藏进度条
router.afterEach(() => {
  NProgress.done()
})
</script>

<template>
  <RouterView />
</template>

<style>
/* 自定义进度条颜色 */
#nprogress .bar {
  background-color: #ff6900 !important;
  height: 3px !important;
}

#nprogress .peg {
  box-shadow: 0 0 10px rgba(255, 105, 0, 0.55), 0 0 5px rgba(255, 105, 0, 0.35) !important;
}
</style>
