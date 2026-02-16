import {
  ConstsCopySetting,
  ConstsWatermarkSetting,
  DomainBrandSettings,
  DomainDisclaimerSettings,
  DomainConversationSetting,
  DomainSEOSettings,
  DomainWebAppLandingConfig,
} from '@/request/types';

export interface NavBtn {
  id: string;
  url: string;
  variant: 'contained' | 'outlined';
  showIcon: boolean;
  icon: string;
  text: string;
  target: '_blank' | '_self';
}

export interface Heading {
  id: string;
  title: string;
  heading: number;
}

export interface FooterSetting {
  footer_style: 'simple' | 'complex';
  corp_name: string;
  icp: string;
  brand_name: string;
  brand_desc: string;
  brand_logo: string;
  brand_groups: BrandGroup[];
}

export interface BrandGroup {
  name: string;
  links: {
    name: string;
    url: string;
  }[];
}

export interface AuthSetting {
  enabled: boolean;
  password?: string;
}

export interface CatalogSetting {
  catalog_visible: 1 | 2;
  catalog_folder: 1 | 2;
  catalog_width: number;
}

export interface ThemeAndStyleSetting {
  bg_image: string;
  doc_width: string;
}

export interface KBDetail {
  name: string;
  base_url?: string;
  settings: {
    conversation_setting: DomainConversationSetting;
    brand_settings?: DomainBrandSettings;
    title: string;
    btns: NavBtn[];
    icon: string;
    language?: string;
    welcome_str: string;
    search_placeholder: string;
    recommend_questions: string[];
    recommend_node_ids: string[];
    desc: string;
    keyword: string;
    seo_settings?: DomainSEOSettings;
    head_code: string;
    body_code: string;
    theme_mode?: 'light' | 'dark';
    simple_auth?: AuthSetting | null;
    footer_settings?: FooterSetting | null;
    catalog_settings?: CatalogSetting | null;
    theme_and_style?: ThemeAndStyleSetting | null;
    watermark_content?: string;
    watermark_setting?: ConstsWatermarkSetting;
    copy_setting?: ConstsCopySetting;
    copy_append_content?: string;
    node_meta_settings?: {
      show_created_at?: boolean;
      show_updated_at?: boolean;
      show_word_count?: boolean;
    };
    security_settings?: {
      allow_widget_image_upload?: boolean;
    };
    disclaimer_settings?: DomainDisclaimerSettings;
    web_app_custom_style: {
      allow_theme_switching?: boolean;
      header_search_placeholder?: string;
      show_brand_info?: boolean;
      social_media_accounts?: DomainSocialMediaAccount[];
      footer_show_intro?: boolean;
    };
    contribute_settings?: {
      is_enable: boolean;
    };
    web_app_landing_configs: DomainWebAppLandingConfig[];
  };
}
export interface DomainSocialMediaAccount {
  channel?: string;
  icon?: string;
  link?: string;
  text?: string;
  phone?: string;
}

export type WidgetInfo = {
  recommend_nodes: RecommendNode[];
  settings: {
    brand_settings?: DomainBrandSettings;
    security_settings?: {
      allow_widget_image_upload?: boolean;
    };
    title: string;
    icon: string;
    language?: string;
    welcome_str: string;
    search_placeholder: string;
    recommend_questions: string[];
    widget_bot_settings: {
      btn_logo?: string;
      btn_text?: string;
      btn_style?: string;
      btn_id?: string;
      btn_position?: string;
      modal_position?: string;
      is_open?: boolean;
      recommend_node_ids?: string[];
      recommend_questions?: string[];
      theme_mode?: string;
      search_mode?: string;
      placeholder?: string;
      disclaimer?: string;
      copyright_hide_enabled?: boolean;
      copyright_info?: string;
    };
  };
};

export type RecommendNode = {
  id: string;
  name: string;
  type: 1 | 2;
  emoji: string;
  parent_id: string;
  summary: string;
  position: number;
  recommend_nodes?: RecommendNode[];
};

export interface NodeDetail {
  id: string;
  kb_id: string;
  name: string;
  content: string;
  created_at: string;
  updated_at: string;
  type: 1 | 2;
  creator_account: string;
  editor_account: string;
  meta: {
    doc_width: string;
    summary: string;
    emoji?: string;
  };
}

export interface NodeListItem {
  id: string;
  name: string;
  type: 1 | 2;
  emoji: string;
  position: number;
  parent_id: string;
  summary: string;
  created_at: string;
  updated_at: string;
  status: 1 | 2; // 1 草稿 2 发布
}

export interface ChunkResultItem {
  node_id: string;
  name: string;
  summary: string;
}

export interface ITreeItem {
  id: string;
  name: string;
  level: number;
  order?: number;
  emoji?: string;
  defaultExpand?: boolean;
  expanded?: boolean;
  parentId?: string | null;
  summary?: string;
  children?: ITreeItem[];
  type: 1 | 2;
  isEditting?: boolean;
  canHaveChildren?: boolean;
  updated_at?: string;
  status?: 1 | 2;
}

export interface ConversationItem {
  q: string;
  a: string;
  score: number;
  update_time: string;
  message_id: string;
  source: 'history' | 'chat';
  chunk_result: ChunkResultItem[];
}
