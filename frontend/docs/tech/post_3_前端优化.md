## 前端性能优化

前端性能优化是提升用户体验的重要环节，尤其是在移动互联网时代，用户对加载速度的要求越来越高。优化前端性能可以从多个方面入手，其中首屏加载时间和代码分割是两个非常关键的点。

### ## 首屏加载

首屏加载时间是指用户打开网页到能看到主要内容的时间。优化首屏加载时间可以大大提高用户体验。一个有效的方法是优化资源加载顺序，确保最先加载的是用户最需要看到的内容。

```html
<!-- 优先加载关键CSS -->
<link rel="stylesheet" href="critical.css" media="screen">
<!-- 非关键CSS异步加载 -->
<link rel="stylesheet" href="non-critical.css" media="print" onload="this.media='screen'">
```

### ## 代码分割

代码分割可以显著减少首次加载的时间，通过将代码分割成多个小块，只在需要时加载，可以大大减少初始加载时间。使用Webpack等模块打包工具可以轻松实现代码分割。

```javascript
// 使用动态导入实现代码分割
import('./chunk.js').then((module) => {
  // 使用模块中的函数
  module.default();
});
```

### ## 性能优化

除了首屏加载和代码分割，还有很多其他技术可以用来优化前端性能，如图片懒加载、服务端渲染（SSR）、使用CDN等。合理的缓存策略也是提升性能的关键。

```javascript
// 图片懒加载示例
const images = document.querySelectorAll('img');
const options = {
  rootMargin: '50px',
  threshold: 0.01
};
const observer = new IntersectionObserver((entries, observer) => {
  entries.forEach(entry => {
    if (entry.isIntersecting) {
      const img = entry.target;
      img.src = img.dataset.src;
      observer.unobserve(img);
    }
  });
}, options);

images.forEach(img => {
  observer.observe(img);
});
```

通过上述方法，我们可以显著提升网站的性能，提供更流畅的用户体验。