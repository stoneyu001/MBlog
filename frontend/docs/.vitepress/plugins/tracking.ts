/**
 * Vitepress 无感埋点插件
 */

import { type Router, EnhanceAppContext, PageData } from 'vitepress'

// 判断是否在浏览器环境运行
const isBrowser = typeof window !== 'undefined'

// 埋点事件类型
export enum TrackEventType {
  PAGEVIEW = 'PAGEVIEW',
  CLICK = 'CLICK',
  EXPOSURE = 'EXPOSURE', // 曝光
  CUSTOM = 'CUSTOM'      // 自定义事件
}

// 埋点事件接口
export interface TrackEvent {
  sessionId?: string;
  userId?: string;
  eventType: TrackEventType | string;
  elementPath?: string;
  pagePath: string;
  referrer?: string;
  metadata?: Record<string, any>;
  timestamp: number;
}

// 埋点配置选项
export interface TrackingOptions {
  endpoint: string;       // 埋点上报接口
  batchSize?: number;     // 批量上报大小 
  batchInterval?: number; // 批量上报间隔(ms)
  debug?: boolean;        // 调试模式
  sampling?: number;      // 采样率 0-1
  excludePaths?: string[]; // 排除的页面路径
  includeElementSelector?: string[]; // 包含的元素选择器
  enableAutoTrack?: {     // 自动埋点配置
    pageview?: boolean;   // 页面访问
    click?: boolean;      // 点击事件
    exposure?: boolean;   // 曝光事件
  };
}

// 设备指纹生成
function generateFingerprint(): string {
  if (!isBrowser) return 'server-side-rendering'
  
  const components = [
    navigator.userAgent,
    screen.width + 'x' + screen.height,
    screen.colorDepth,
    new Date().getTimezoneOffset(),
    navigator.language,
    navigator.hardwareConcurrency || '',
    navigator.platform || '',
  ];
  
  return hashCode(components.join('|')).toString(16);
}

// 简单的哈希函数
function hashCode(str: string): number {
  let hash = 0;
  if (str.length === 0) return hash;
  
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash; // Convert to 32bit integer
  }
  
  return Math.abs(hash);
}

export class Tracker {
  private options: TrackingOptions;
  private events: TrackEvent[] = [];
  private timer: ReturnType<typeof setTimeout> | null = null;
  private sessionId: string;
  private userId: string | null = null;
  
  constructor(options: TrackingOptions) {
    // 默认配置
    this.options = {
      batchSize: 10,
      batchInterval: 5000, // 5秒
      debug: false,
      sampling: 1,
      enableAutoTrack: {
        pageview: true,
        click: true,
        exposure: false
      },
      ...options
    };
    
    // 生成会话ID
    this.sessionId = this.getOrCreateSessionId();
    
    // 发送缓冲区内事件
    this.startBatchTimer();
  }
  
  // 获取或创建会话ID
  private getOrCreateSessionId(): string {
    // 如果不在浏览器环境，返回固定ID
    if (!isBrowser) return 'server-side-rendering';
    
    const storageKey = 'track_session_id';
    let sessionId: string | null = null;
    
    try {
      sessionId = localStorage.getItem(storageKey);
    } catch (e) {
      // 处理localStorage不可用的情况
      return 'storage-unavailable';
    }
    
    if (!sessionId) {
      sessionId = generateFingerprint();
      try {
        localStorage.setItem(storageKey, sessionId);
      } catch (e) {
        // 忽略写入错误
      }
    }
    
    return sessionId;
  }
  
  // 设置用户ID
  public setUserId(userId: string): void {
    this.userId = userId;
  }
  
  // 追踪事件
  public track(event: Omit<TrackEvent, 'sessionId' | 'userId' | 'timestamp'>): void {
    // 如果不在浏览器环境，不执行埋点
    if (!isBrowser) return;
    
    // 采样判断
    if (Math.random() > (this.options.sampling || 1)) {
      this.log('事件因采样被丢弃', event);
      return;
    }
    
    // 排除路径判断
    if (this.shouldExcludePath(event.pagePath)) {
      this.log('事件路径被排除', event);
      return;
    }
    
    const fullEvent: TrackEvent = {
      sessionId: this.sessionId,
      userId: this.userId || undefined,
      timestamp: Date.now(),
      ...event
    };
    
    this.events.push(fullEvent);
    this.log('事件已追踪', fullEvent);
    
    // 如果达到批处理大小，立即发送
    if (this.events.length >= (this.options.batchSize || 10)) {
      this.flush();
    }
  }
  
  // 判断是否应该排除该路径
  private shouldExcludePath(path: string): boolean {
    if (!this.options.excludePaths || this.options.excludePaths.length === 0) {
      return false;
    }
    
    return this.options.excludePaths.some(pattern => {
      if (pattern.includes('*')) {
        const regexPattern = pattern.replace(/\*/g, '.*');
        return new RegExp(regexPattern).test(path);
      }
      return path === pattern;
    });
  }
  
  // 页面访问埋点
  public trackPageView(path: string, referrer?: string): void {
    if (!isBrowser) return;
    
    this.track({
      eventType: TrackEventType.PAGEVIEW,
      pagePath: path,
      referrer: referrer || (typeof document !== 'undefined' ? document.referrer : ''),
      metadata: {
        title: typeof document !== 'undefined' ? document.title : '',
        url: typeof window !== 'undefined' ? window.location.href : ''
      }
    });
  }
  
  // 点击埋点
  public trackClick(element: HTMLElement, path: string): void {
    if (!isBrowser) return;
    
    const elementPath = this.getElementPath(element);
    this.track({
      eventType: TrackEventType.CLICK,
      pagePath: path,
      elementPath: elementPath,
      metadata: {
        text: element.textContent?.trim().substring(0, 50) || '',
        tagName: element.tagName.toLowerCase(),
        className: element.className,
        id: element.id
      }
    });
  }
  
  // 获取元素路径
  private getElementPath(element: HTMLElement, maxDepth: number = 5): string {
    const path: string[] = [];
    let currentElement: HTMLElement | null = element;
    let depth = 0;
    
    while (currentElement && depth < maxDepth) {
      let identifier = currentElement.tagName.toLowerCase();
      
      if (currentElement.id) {
        identifier += `#${currentElement.id}`;
      } else if (currentElement.className) {
        const classList = currentElement.className.split(/\s+/).filter(Boolean);
        if (classList.length > 0) {
          identifier += `.${classList.join('.')}`;
        }
      }
      
      // 添加位置索引
      if (currentElement.parentElement) {
        const siblings = Array.from(currentElement.parentElement.children);
        const index = siblings.indexOf(currentElement);
        if (index !== -1) {
          identifier += `:nth-child(${index + 1})`;
        }
      }
      
      path.unshift(identifier);
      currentElement = currentElement.parentElement;
      depth++;
    }
    
    return path.join(' > ');
  }
  
  // 刷新缓冲区，发送事件
  public flush(): void {
    if (!isBrowser || this.events.length === 0) {
      return;
    }
    
    const eventsToSend = [...this.events];
    this.events = [];
    
    this.sendEvents(eventsToSend)
      .then(() => {
        this.log(`成功发送 ${eventsToSend.length} 个事件`);
      })
      .catch(error => {
        this.log('发送事件失败', error);
        // 重新加入队列，优先发送
        this.events = [...eventsToSend, ...this.events];
      });
  }
  
  // 发送事件到服务器
  private async sendEvents(events: TrackEvent[]): Promise<void> {
    try {
      const response = await fetch(this.options.endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ events }),
        keepalive: true // 允许页面关闭时仍可发送请求
      });
      
      if (!response.ok) {
        throw new Error(`HTTP error ${response.status}`);
      }
    } catch (error) {
      this.log('发送事件失败', error);
      throw error;
    }
  }
  
  // 设置定时发送
  private startBatchTimer(): void {
    if (!isBrowser) return;
    
    this.timer = setInterval(() => {
      if (this.events.length > 0) {
        this.flush();
      }
    }, this.options.batchInterval || 5000);
  }
  
  // 停止定时器
  public dispose(): void {
    if (!isBrowser) return;
    
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
    
    // 尝试发送剩余事件
    if (this.events.length > 0) {
      this.flush();
    }
  }
  
  // 调试日志
  private log(...args: any[]): void {
    if (isBrowser && this.options.debug) {
      console.log('[Tracker]', ...args);
    }
  }
  
  // 设置全局点击监听
  public setupClickTracking(router: Router): void {
    if (!isBrowser || !this.options.enableAutoTrack?.click) {
      return;
    }
    
    document.addEventListener('click', (event) => {
      const target = event.target as HTMLElement;
      if (!target) return;
      
      // 安全地获取路由路径
      const currentPath = router && router.route && router.route.path 
        ? router.route.path 
        : window.location.pathname;
        
      // 判断元素是否应该被跟踪
      if (this.shouldTrackElement(target) && currentPath) {
        this.trackClick(target, currentPath);
      }
    }, { passive: true, capture: true });
  }
  
  // 判断元素是否应该被跟踪
  private shouldTrackElement(element: HTMLElement): boolean {
    // 排除明确标记为不跟踪的元素
    if (element.hasAttribute('data-track-ignore')) {
      return false;
    }
    
    // 检查是否是表单敏感元素
    const sensitiveElements = ['input', 'textarea', 'select', 'password'];
    if (sensitiveElements.includes(element.tagName.toLowerCase())) {
      // 检查是否是密码输入
      if (element.tagName.toLowerCase() === 'input' &&
          (element as HTMLInputElement).type === 'password') {
        return false;
      }
    }
    
    // 如果设置了包含选择器，检查是否匹配
    if (this.options.includeElementSelector && this.options.includeElementSelector.length > 0) {
      for (const selector of this.options.includeElementSelector) {
        if (element.matches(selector)) {
          return true;
        }
      }
      
      // 寻找符合选择器的父元素
      let parent = element.parentElement;
      while (parent) {
        for (const selector of this.options.includeElementSelector) {
          if (parent.matches(selector)) {
            return true;
          }
        }
        parent = parent.parentElement;
      }
      
      return false;
    }
    
    // 跟踪可点击元素
    const clickableElements = ['a', 'button', '[role="button"]', '[role="link"]', '[role="menuitem"]'];
    for (const selector of clickableElements) {
      if (element.matches(selector)) {
        return true;
      }
    }
    
    return false;
  }
}

// 创建VitePress埋点插件
export function createTrackingPlugin(options: TrackingOptions) {
  return {
    tracker: null as Tracker | null,
    
    install(router: Router) {
      // 只在浏览器环境下创建跟踪器
      if (!isBrowser) return;
      
      // 创建跟踪器实例
      this.tracker = new Tracker(options);
      
      if (options.enableAutoTrack?.pageview) {
        // 监听路由变化
        router.onAfterRouteChanged = (to: any) => {
          // 安全地获取当前和目标路由路径
          const toPath = typeof to === 'object' && to && 'path' in to ? String(to.path) : '';
          const fromPath = router.route && typeof router.route === 'object' && 'path' in router.route 
            ? String(router.route.path) 
            : '';
            
          if (this.tracker && toPath) {
            this.tracker.trackPageView(toPath, fromPath);
          }
        };
        
        // 初始页面访问
        const currentPath = router.route && typeof router.route === 'object' && 'path' in router.route 
          ? String(router.route.path) 
          : '';
          
        if (this.tracker && currentPath) {
          this.tracker.trackPageView(currentPath);
        }
      }
      
      if (options.enableAutoTrack?.click && this.tracker) {
        // 设置点击跟踪
        this.tracker.setupClickTracking(router);
      }
      
      // 添加到window对象，方便全局使用
      if (isBrowser) {
        (window as any).__tracker = this.tracker;
      }
    },
    
    dispose() {
      if (this.tracker) {
        this.tracker.dispose();
        this.tracker = null;
      }
    }
  };
}

export default createTrackingPlugin; 