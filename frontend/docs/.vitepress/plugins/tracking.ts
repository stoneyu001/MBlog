/**
 * Vitepress 无感埋点插件
 */

import { type Router, EnhanceAppContext, PageData } from 'vitepress'
import { Logger } from './utils/logger'
import { detectPlatform } from './utils/platform'
import { getOrCreateFingerprint } from './utils/fingerprint'
import {
  generateSessionId,
  getSessionData,
  saveSessionData,
  updateSessionActivity,
  type SessionData
} from './utils/session'

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
  // 核心字段：事件类型和页面路径
  event_type: TrackEventType | string;
  page_path: string;

  // 会话相关字段（可选，由系统补充）
  session_id?: string;
  user_id?: string;
  timestamp?: number;

  // 事件详细信息（可选）
  element_path?: string;
  referrer?: string;
  platform?: string;
  event_duration?: number;

  // 扩展信息
  metadata?: Record<string, any>;
}

// SessionData接口已从 utils/session 导入

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

  try {
    console.log('[Tracker Debug] 开始生成设备指纹');

    // 收集更多设备信息提高唯一性
    const components = [
      navigator.userAgent || 'unknown',
      `${screen.width || 0}x${screen.height || 0}`,
      screen.colorDepth || 0,
      new Date().getTimezoneOffset(),
      navigator.language || 'unknown',
      navigator.hardwareConcurrency || 0,
      navigator.platform || 'unknown',
      navigator.cookieEnabled ? '1' : '0',
      navigator.doNotTrack || 'unknown',
      // 添加更多唯一标识
      navigator.vendor || 'unknown',
      navigator.productSub || 'unknown',
      navigator.maxTouchPoints || 0,
      // 添加内存信息（如果可用）
      // @ts-ignore
      navigator.deviceMemory || 'unknown',
      // 添加连接信息（如果可用）
      // @ts-ignore
      navigator.connection?.type || 'unknown',
      // 添加随机种子
      Math.random().toString(36).substring(2) + Date.now().toString(36)
    ];

    console.log('[Tracker Debug] 收集的设备信息:', components);

    // 生成指纹
    const rawFingerprint = components.join('|');
    console.log('[Tracker Debug] 原始指纹字符串:', rawFingerprint);

    const hash = hashCode(rawFingerprint);
    console.log('[Tracker Debug] 哈希值:', hash);

    // 确保生成的指纹是正数且为16进制
    const fingerprint = Math.abs(hash).toString(16);
    console.log('[Tracker Debug] 最终设备指纹:', fingerprint);

    // 验证指纹有效性
    if (!fingerprint || fingerprint.length < 8) {
      throw new Error('生成的指纹无效');
    }

    return fingerprint;
  } catch (error) {
    console.error('[Tracker Error] 生成设备指纹失败:', error);

    // 使用备用方案生成指纹
    const timestamp = Date.now();
    const random = Math.random();
    const fallbackComponents = [
      timestamp.toString(36),
      random.toString(36).substring(2),
      navigator.userAgent || 'unknown'
    ];

    const fallbackFingerprint = hashCode(fallbackComponents.join('|')).toString(16);
    console.log('[Tracker Debug] 使用备用指纹:', fallbackFingerprint);

    return fallbackFingerprint;
  }
}

// 改进的哈希函数
function hashCode(str: string): number {
  let hash = 0;
  if (!str || str.length === 0) return 1;

  // 使用FNV-1a哈希算法
  const FNV_PRIME = 16777619;
  const FNV_OFFSET_BASIS = 2166136261;

  hash = FNV_OFFSET_BASIS;
  for (let i = 0; i < str.length; i++) {
    hash ^= str.charCodeAt(i);
    hash *= FNV_PRIME;
  }

  // 确保返回正数
  return Math.abs(hash || 1);
}

// 生成随机会话ID
function generate_session_id(): string {
  return Date.now().toString(36) + Math.random().toString(36).substring(2);
}

export class Tracker {
  private options: TrackingOptions;
  private events: TrackEvent[] = [];
  private timer: ReturnType<typeof setTimeout> | null = null;
  private session_id: string;
  private device_fingerprint: string;
  private readonly SESSION_STORAGE_KEY = 'track_session_data';
  private readonly FINGERPRINT_STORAGE_KEY = 'track_device_fingerprint';
  private pageEnterTime: number = 0;

  constructor(options: TrackingOptions) {
    // 默认配置
    this.options = {
      batchSize: 10,
      batchInterval: 5000, // 5秒
      debug: true, // 默认开启调试
      sampling: 1,
      enableAutoTrack: {
        pageview: true,
        click: true,
        exposure: false
      },
      ...options
    };

    // 添加调试日志
    this.log('Tracker初始化 - 开始');

    // 获取或生成设备指纹
    this.device_fingerprint = this.getOrCreateFingerprint();
    this.log('设备指纹:', this.device_fingerprint);

    // 始终生成新的会话ID
    this.session_id = this.createNewSession();
    this.log('新会话ID:', this.session_id);

    // 发送缓冲区内事件
    this.startBatchTimer();

    this.log('Tracker初始化 - 完成');
  }

  // 获取或创建设备指纹
  private getOrCreateFingerprint(): string {
    if (!isBrowser) return 'server-side-rendering';

    try {
      console.log('[Tracker Debug] 开始获取设备指纹');

      // 尝试从localStorage获取存储的设备指纹
      const storedFingerprint = localStorage.getItem(this.FINGERPRINT_STORAGE_KEY);
      console.log('[Tracker Debug] 存储的设备指纹:', storedFingerprint);

      if (storedFingerprint && storedFingerprint.length >= 8) {
        console.log('[Tracker Debug] 使用已存储的设备指纹');
        return storedFingerprint;
      }

      // 生成新的设备指纹
      console.log('[Tracker Debug] 生成新的设备指纹');
      const fingerprint = generateFingerprint();

      // 验证生成的指纹
      if (!fingerprint || fingerprint.length < 8) {
        throw new Error('生成的设备指纹无效');
      }

      // 验证localStorage是否可用
      const testKey = '_test_storage_';
      try {
        localStorage.setItem(testKey, '1');
        localStorage.removeItem(testKey);
      } catch (e) {
        console.error('[Tracker Error] localStorage不可用:', e);
        return fingerprint; // 直接返回生成的指纹
      }

      // 将指纹存储到localStorage
      try {
        localStorage.setItem(this.FINGERPRINT_STORAGE_KEY, fingerprint);
        console.log('[Tracker Debug] 设备指纹已保存到localStorage');
      } catch (e) {
        console.error('[Tracker Error] 保存设备指纹失败:', e);
      }

      return fingerprint;
    } catch (e) {
      console.error('[Tracker Error] 获取/创建设备指纹时出错:', e);
      // 生成临时指纹
      const tempFingerprint = Date.now().toString(36) + Math.random().toString(36).substring(2);
      return tempFingerprint;
    }
  }

  // 创建新会话
  private createNewSession(): string {
    if (!isBrowser) return 'server-side-rendering';

    // 检查是否存在现有会话（仅在同一浏览器标签页的情况下）
    const existing_session_id = this.checkExistingSession();
    if (existing_session_id) {
      this.log('使用现有会话:', existing_session_id);
      return existing_session_id;
    }

    // 生成新的会话ID
    const session_id = generate_session_id();
    this.log('生成新会话ID:', session_id);

    try {
      // 将会话ID保存到sessionStorage（浏览器标签页关闭后自动清除）
      sessionStorage.setItem(this.SESSION_STORAGE_KEY, session_id);

      // 保存会话创建时间，用于同一会话内的判断
      const session_data = {
        id: session_id,
        fingerprint: this.device_fingerprint,
        created: Date.now(),
        last_activity: Date.now()
      };

      // 使用sessionStorage而非localStorage，确保标签页关闭后会重新创建
      sessionStorage.setItem(this.SESSION_STORAGE_KEY + '_data', JSON.stringify(session_data));

      this.log('创建新会话', session_data);
    } catch (e) {
      this.log('保存会话数据失败', e);
    }

    return session_id;
  }

  // 检查并可能恢复现有会话（仅在同一浏览器会话内）
  private checkExistingSession(): string | null {
    if (!isBrowser) return null;

    try {
      // 从sessionStorage获取当前会话ID
      return sessionStorage.getItem(this.SESSION_STORAGE_KEY);
    } catch (e) {
      this.log('检查现有会话失败', e);
      return null;
    }
  }

  // 更新会话活动时间
  private updateSessionActivity(): void {
    if (!isBrowser) return;

    try {
      const session_data_str = sessionStorage.getItem(this.SESSION_STORAGE_KEY + '_data');
      if (session_data_str) {
        const session_data: SessionData = JSON.parse(session_data_str);
        session_data.last_activity = Date.now();
        sessionStorage.setItem(this.SESSION_STORAGE_KEY + '_data', JSON.stringify(session_data));
      }
    } catch (e) {
      this.log('更新会话活动时间失败', e);
    }
  }

  // 追踪事件
  public track(event: Omit<TrackEvent, 'session_id' | 'user_id' | 'timestamp'>): void {
    // 如果不在浏览器环境，不执行埋点
    if (!isBrowser) return;

    try {
      // 更新会话活动时间
      this.updateSessionActivity();

      // 采样判断
      if (Math.random() > (this.options.sampling || 1)) {
        this.log('事件因采样被丢弃', event);
        return;
      }

      // 排除路径判断
      if (this.shouldExcludePath(event.page_path)) {
        this.log('事件路径被排除', event);
        return;
      }

      // 获取平台信息
      let platform = event.platform;
      try {
        if (!platform) {
          platform = this.getPlatformInfo();
          this.log('获取平台信息成功:', platform);
        }
      } catch (error) {
        this.log('获取平台信息失败:', error);
        platform = 'unknown';
      }

      // 确保使用最新的会话和设备信息
      this.log(`构建事件数据: session_id=${this.session_id}, user_id=${this.device_fingerprint}, platform=${platform}`);

      // 构建完整事件对象
      const fullEvent: TrackEvent = {
        // 基本事件信息
        event_type: event.event_type,
        page_path: event.page_path,

        // 会话信息
        session_id: this.session_id,
        user_id: this.device_fingerprint,
        timestamp: Date.now(),

        // 可选字段
        element_path: event.element_path,
        referrer: event.referrer,
        platform: platform,
        event_duration: event.event_duration || 0,

        // 元数据
        metadata: {
          ...event.metadata,
          platform_info: platform,  // 在元数据中也保存平台信息
          client_timestamp: Date.now()  // 添加客户端时间戳
        }
      };

      this.events.push(fullEvent);
      this.log('事件已追踪', fullEvent);

      // 如果达到批处理大小，立即发送
      if (this.events.length >= (this.options.batchSize || 10)) {
        this.flush();
      }
    } catch (error) {
      // 即使发生错误，也尝试发送基本事件信息
      this.log('事件处理发生错误，尝试发送基本信息:', error);
      const basicEvent: TrackEvent = {
        event_type: event.event_type,
        page_path: event.page_path,
        session_id: this.session_id,
        user_id: this.device_fingerprint,
        timestamp: Date.now(),
        metadata: {
          error: error instanceof Error ? error.message : 'Unknown error',
          original_event: JSON.stringify(event)
        }
      };
      this.events.push(basicEvent);
    }
  }

  // 获取平台信息
  private getPlatformInfo(): string {
    if (!isBrowser) return 'server-side';

    try {
      const ua = navigator.userAgent.toLowerCase();
      this.log('User Agent:', ua);

      // 基本操作系统检测
      let os = 'unknown';
      if (/windows/.test(ua)) {
        os = 'Windows';
      } else if (/macintosh|mac os x/.test(ua)) {
        os = 'macOS';
      } else if (/linux/.test(ua)) {
        os = 'Linux';
      } else if (/android/.test(ua)) {
        os = 'Android';
      } else if (/iphone|ipad|ipod/.test(ua)) {
        os = 'iOS';
      }

      // 基本浏览器检测（按优先级排序）
      let browser = 'unknown';
      if (/edg\/|edge\//.test(ua)) {
        browser = 'Edge';
      } else if (/chrome\//.test(ua) && !/edg\/|edge\//.test(ua)) {
        browser = 'Chrome';
      } else if (/firefox\//.test(ua)) {
        browser = 'Firefox';
      } else if (/safari\//.test(ua) && !/chrome\//.test(ua)) {
        browser = 'Safari';
      } else if (/opera|opr\//.test(ua)) {
        browser = 'Opera';
      }

      // 返回简单的平台信息
      const platformInfo = `${os}/${browser}`;
      this.log(`检测到的平台信息: OS=${os}, Browser=${browser}`);
      return platformInfo;

    } catch (error) {
      this.log('平台检测出错:', error);
      return 'unknown/unknown';
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
  public trackPageView(path: string, referrer?: string, extraMetadata?: Record<string, any>): void {
    if (!isBrowser) return;

    const encodedPath = encodeURIComponent(path);
    const encodedReferrer = referrer ? encodeURIComponent(referrer) : '';

    const now = Date.now();

    // 计算持续时间（转换为秒）
    let event_duration = 0;
    if (this.pageEnterTime > 0) {
      const durationMs = Math.max(0, now - this.pageEnterTime);
      event_duration = Math.floor(durationMs / 1000); // 转换为秒
      this.log(`页面停留时间计算:
        当前时间: ${now}
        进入时间: ${this.pageEnterTime}
        停留时间: ${durationMs}ms (${event_duration}秒)
      `);
    } else {
      this.log('页面首次加载，无法计算停留时间');
    }

    // 更新页面进入时间
    this.pageEnterTime = now;
    this.log(`更新页面进入时间: ${this.pageEnterTime}`);

    // 获取平台信息
    const platform = this.getPlatformInfo();
    this.log(`发送埋点数据: platform=${platform}, duration=${event_duration}秒, path=${encodedPath}`);

    // 构建元数据，包含上一次的时间戳
    const metadata = {
      title: typeof document !== 'undefined' ? document.title : '',
      url: typeof window !== 'undefined' ? encodeURIComponent(window.location.href) : '',
      prev_timestamp: this.pageEnterTime,
      current_timestamp: now,
      duration_ms: event_duration * 1000, // 保存毫秒值在metadata中，用于调试
      platform_info: platform,
      ...extraMetadata
    };

    this.track({
      event_type: TrackEventType.PAGEVIEW,
      page_path: encodedPath,
      referrer: encodedReferrer,
      event_duration,  // 已经是秒
      platform,
      metadata
    });
  }

  // 点击埋点
  public trackClick(element: HTMLElement, path: string): void {
    if (!isBrowser) return;

    const element_path = this.getElementPath(element);
    const encodedPath = encodeURIComponent(path);
    const encodedElementPath = encodeURIComponent(element_path);

    // 获取平台信息
    const platform = this.getPlatformInfo();

    this.track({
      event_type: TrackEventType.CLICK,
      page_path: encodedPath,
      element_path: encodedElementPath,
      platform,
      event_duration: 0,
      metadata: {
        text: element.textContent?.trim().substring(0, 50) || '',
        tagName: element.tagName.toLowerCase(),
        className: element.className,
        id: element.id,
        href: element.tagName.toLowerCase() === 'a' ? encodeURIComponent((element as HTMLAnchorElement).href || '') : undefined,
        platform_info: platform
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
        identifier += `#${encodeURIComponent(currentElement.id)}`;
      } else if (currentElement.className) {
        const classList = currentElement.className.split(/\s+/).filter(Boolean);
        if (classList.length > 0) {
          identifier += `.${classList.map(c => encodeURIComponent(c)).join('.')}`;
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
    // 确保所有URL相关字段都已正确编码
    const processedEvents = events.map(event => {
      // 确保platform存在且不为unknown
      let platform = event.platform;
      if (!platform || platform === 'unknown') {
        platform = this.getPlatformInfo();
        this.log('重新获取平台信息:', platform);
      }

      // 确保event_duration是数字类型
      const event_duration = typeof event.event_duration === 'number' ? event.event_duration : 0;

      this.log(`处理事件: type=${event.event_type}, duration=${event_duration}ms, platform=${platform}`);

      return {
        ...event,
        platform,
        event_duration,
        page_path: event.page_path,
        element_path: event.element_path,
        referrer: event.referrer,
        metadata: event.metadata ? {
          ...event.metadata,
          url: event.metadata.url ? encodeURIComponent(event.metadata.url) : undefined,
          platform_detail: platform  // 添加详细的平台信息到元数据中
        } : { platform_detail: platform }
      };
    });

    this.log('发送数据:', processedEvents);

    try {
      const response = await fetch(this.options.endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Session-ID': this.session_id,
          'X-Device-Fingerprint': this.device_fingerprint
        },
        body: JSON.stringify(processedEvents),
        keepalive: true
      });

      if (!response.ok) {
        throw new Error(`HTTP error ${response.status}`);
      }

      const responseText = await response.text();
      this.log('服务器响应:', responseText);

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

    // 记录最后一个页面的访问时长
    if (this.pageEnterTime > 0) {
      const now = Date.now();
      const lastPageDurationMs = now - this.pageEnterTime;
      const lastPageDuration = Math.floor(lastPageDurationMs / 1000); // 转换为秒
      const currentPath = typeof window !== 'undefined' ? window.location.pathname : '';

      if (currentPath) {
        this.log(`记录最后一个页面的停留时间: ${lastPageDurationMs}ms (${lastPageDuration}秒)`);

        this.track({
          event_type: TrackEventType.PAGEVIEW,
          page_path: encodeURIComponent(currentPath),
          event_duration: lastPageDuration,  // 已经是秒
          platform: this.getPlatformInfo(),
          metadata: {
            is_last_page: true,
            duration_ms: lastPageDurationMs,  // 保存毫秒值在metadata中，用于调试
            exit_timestamp: now
          }
        });
      }
    }

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
    // 无论debug设置如何，都记录重要操作
    if (isBrowser) {
      if (this.options.debug) {
        console.log('%c[Tracker]', 'color: #4CAF50; font-weight: bold;', ...args);
      } else if (args[0]?.startsWith && args[0].startsWith('错误')) {
        // 即使未开启debug，错误信息也会记录
        console.error('%c[Tracker Error]', 'color: #F44336; font-weight: bold;', ...args);
      }
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
      if (!isBrowser) return;

      this.tracker = new Tracker(options);

      if (options.enableAutoTrack?.pageview) {
        // 1. 路由变化时触发
        router.onAfterRouteChanged = (to: any) => {
          const toPath = typeof to === 'object' && to && 'path' in to ? String(to.path) : '';
          const fromPath = router.route && typeof router.route === 'object' && 'path' in router.route
            ? String(router.route.path)
            : '';

          if (this.tracker && toPath) {
            this.tracker.trackPageView(toPath, fromPath);
          }
        };

        // 2. 初始页面加载时触发
        const currentPath = router.route && typeof router.route === 'object' && 'path' in router.route
          ? String(router.route.path)
          : '';

        if (this.tracker && currentPath) {
          this.tracker.trackPageView(currentPath);
        }

        // 3. 添加 History API 监听
        if (typeof window !== 'undefined') {
          // 监听 popstate 事件（浏览器前进/后退）
          window.addEventListener('popstate', () => {
            if (this.tracker) {
              this.tracker.trackPageView(window.location.pathname, document.referrer);
            }
          });

          // 重写 pushState 和 replaceState
          const originalPushState = history.pushState;
          const originalReplaceState = history.replaceState;
          const tracker = this.tracker; // 捕获 tracker 引用

          history.pushState = function (...args) {
            originalPushState.apply(this, args);
            if (tracker) {
              tracker.trackPageView(window.location.pathname, document.referrer);
            }
          };

          history.replaceState = function (...args) {
            originalReplaceState.apply(this, args);
            if (tracker) {
              tracker.trackPageView(window.location.pathname, document.referrer);
            }
          };
        }

        // 4. 添加可视性变化监听
        if (typeof document !== 'undefined') {
          document.addEventListener('visibilitychange', () => {
            if (document.visibilityState === 'visible' && this.tracker) {
              this.tracker.trackPageView(window.location.pathname, document.referrer, {
                visibility_change: true
              });
            }
          });
        }
      }

      if (options.enableAutoTrack?.click && this.tracker) {
        this.tracker.setupClickTracking(router);
      }

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