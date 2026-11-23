type SSECallback<T> = (data: T) => void;
type SSEErrorCallback = (error: Error) => void;
type SSECompleteCallback = () => void;

export class SSEHttpError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = 'SSEHttpError';
    this.status = status;
  }
}

interface SSEClientOptions {
  url: string;
  headers?: Record<string, string>;
  onOpen?: SSECompleteCallback;
  onError?: SSEErrorCallback;
  onCancel?: SSEErrorCallback;
  onComplete?: SSECompleteCallback;
  method?: string;
  /** 整体请求超时时间（毫秒），不传默认 5 分钟 */
  timeout?: number;
  /** 响应模式：'sse'（默认）按 SSE data: 帧解析 JSON；'raw' 将响应文本块原样回调 */
  responseMode?: 'sse' | 'raw';
}

class SSEClient<T> {
  private controller: AbortController;
  private reader: ReadableStreamDefaultReader<Uint8Array> | null;
  private textDecoder: TextDecoder;
  private buffer: string;

  constructor(private options: SSEClientOptions) {
    this.controller = new AbortController();
    this.reader = null;
    this.textDecoder = new TextDecoder();
    this.buffer = '';
  }

  public subscribe(body: BodyInit, onMessage: SSECallback<T>) {
    this.controller.abort();
    this.controller = new AbortController();
    this.buffer = '';
    this.textDecoder = new TextDecoder();
    const {
      url,
      headers,
      onOpen,
      onError,
      onComplete,
      method = 'POST',
      timeout: timeoutDuration = 300000,
    } = this.options;
    const timeoutId = setTimeout(() => {
      this.unsubscribe();
      onError?.(
        new Error(
          `Request timed out after ${Math.round(timeoutDuration / 1000)} seconds`,
        ),
      );
    }, timeoutDuration);

    const upperMethod = method.toUpperCase();
    const hasBody =
      upperMethod !== 'GET' &&
      upperMethod !== 'HEAD' &&
      body !== undefined &&
      body !== null;

    const docLang =
      typeof document !== 'undefined' ? document.documentElement?.lang : '';

    fetch(url, {
      method,
      headers: {
        Accept: 'text/event-stream',
        ...(hasBody ? { 'Content-Type': 'application/json' } : {}),
        ...(docLang ? { 'X-Lang': docLang, 'Accept-Language': docLang } : {}),
        ...headers,
      },
      body: hasBody ? body : undefined,
      signal: this.controller.signal,
    })
      .then(async response => {
        if (!response.ok) {
          clearTimeout(timeoutId);
          let errorMessage = `HTTP error! status: ${response.status}`;
          try {
            const body = await response.json();
            if (body?.message) errorMessage = body.message;
            else if (body?.error) errorMessage = body.error;
          } catch {}
          throw new SSEHttpError(response.status, errorMessage);
        }
        if (!response.body) {
          clearTimeout(timeoutId);
          onError?.(new Error('No response body'));
          return;
        }

        onOpen?.();
        this.reader = response.body.getReader();

        while (true) {
          const { done, value } = await this.reader.read();
          if (done) {
            clearTimeout(timeoutId);
            onComplete?.();
            break;
          }
          this.processChunk(value, onMessage);
        }
      })
      .catch(error => {
        clearTimeout(timeoutId);
        if (error.name !== 'AbortError') {
          onError?.(error);
        }
      });
  }

  private processChunk(
    chunk: Uint8Array | undefined,
    callback: SSECallback<T>,
  ) {
    if (!chunk) return;

    const text = this.textDecoder.decode(chunk, { stream: true });
    if (this.options.responseMode === 'raw') {
      callback(text as T);
      return;
    }

    this.buffer += text;
    const lines = this.buffer.split('\n');
    let currentData = '';
    let isDataLine = false;

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i].trim();
      if (line.startsWith('data: ')) {
        if (isDataLine) {
          currentData += '\n';
        }
        currentData += line.slice(6);
        isDataLine = true;
      } else if (line === '') {
        if (isDataLine) {
          try {
            const data = JSON.parse(currentData) as T;
            callback(data);
          } catch (error) {
            console.error(error);
            this.options.onError?.(new Error('Failed to parse SSE data'));
          }
          currentData = '';
          isDataLine = false;
        }
      }
    }

    this.buffer = lines[lines.length - 1];
  }

  public unsubscribe() {
    this.controller.abort();
    if (this.reader) {
      this.reader.cancel().catch(() => {});
    }
    this.options.onCancel?.(new Error('Request canceled'));
  }
}

export default SSEClient;
