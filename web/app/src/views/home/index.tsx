'use client';

import { Banner } from '@panda-wiki/ui';
import dynamic from 'next/dynamic';
import { DomainRecommendNodeListResp } from '@/request/types';

import { useStore } from '@/provider';
import { useBasePath } from '@/hooks';
import { getImagePath } from '@/utils/getImagePath';
import { useI18n } from '@/i18n/useI18n';

const handleFaqProps = (config: any = {}, labels: HomeLabels) => {
  return {
    title: config.title || labels.linkGroup,
    items:
      config.list?.map((item: any) => ({
        question: item.question,
        url: item.link,
      })) || [],
  };
};

const handleBasicDocProps = (
  config: any = {},
  docs: DomainRecommendNodeListResp[],
  basePath: string,
  labels: HomeLabels,
) => {
  return {
    title: config.title || labels.docSummaryCard,
    basePath,
    items:
      docs?.map(item => ({
        ...item,
        summary: item.summary || labels.noSummary,
      })) || [],
  };
};

const handleDirDocProps = (
  config: any = {},
  docs: DomainRecommendNodeListResp[],
  basePath: string,
  labels: HomeLabels,
) => {
  return {
    title: config.title || labels.docDirCard,
    viewMoreLabel: labels.viewMore,
    basePath,
    items:
      docs?.map(item => ({
        id: item.id,
        name: item.name,
        ...item,
        recommend_nodes: [...(item.recommend_nodes || [])].sort(
          (a, b) => (a.position ?? 0) - (b.position ?? 0),
        ),
      })) || [],
  };
};

const handleSimpleDocProps = (
  config: any = {},
  docs: DomainRecommendNodeListResp[],
  basePath: string,
  labels: HomeLabels,
) => {
  return {
    title: config.title || labels.simpleDocCard,
    basePath,
    items:
      docs?.map(item => ({
        ...item,
      })) || [],
  };
};

const handleCarouselProps = (
  config: any = {},
  basePath: string,
  labels: HomeLabels,
) => {
  return {
    title: config.title || labels.carousel,
    items:
      config.list?.map((item: any) => ({
        id: item.id,
        title: item.title,
        url: getImagePath(item.url, basePath),
        desc: item.desc,
      })) || [],
  };
};

const handleBannerProps = (
  config: any = {},
  basePath: string,
  labels: HomeLabels,
) => {
  return {
    title: {
      text: config.title,
    },
    subtitle: {
      text: config.subtitle,
    },
    bg_url: getImagePath(config.bg_url, basePath),
    search: {
      placeholder: config.placeholder,
      hot: config.hot_search,
    },
    chatLabel: labels.chatTab,
  };
};

const handleTextProps = (config: any = {}, labels: HomeLabels) => {
  return {
    title: config.title || labels.title,
  };
};

const handleCaseProps = (config: any = {}, labels: HomeLabels) => {
  return {
    title: config.title || labels.case,
    items: config.list || [],
  };
};

const handleMetricsProps = (config: any = {}, labels: HomeLabels) => {
  return {
    title: config.title || labels.metrics,
    items: config.list || [],
  };
};

const handleFeatureProps = (config: any = {}, labels: HomeLabels) => {
  return {
    title: config.title || labels.feature,
    items: config.list || [],
  };
};

const handleImgTextProps = (
  config: any = {},
  basePath: string,
  labels: HomeLabels,
) => {
  return {
    title: config.title || labels.leftImgRightText,
    item: {
      ...config.item,
      url: getImagePath(config.item?.url, basePath),
    },
    direction: 'row',
  };
};

const handleTextImgProps = (
  config: any = {},
  basePath: string,
  labels: HomeLabels,
) => {
  return {
    title: config.title || labels.rightImgLeftText,
    item: {
      ...config.item,
      url: getImagePath(config.item?.url, basePath),
    },
    direction: 'row-reverse',
  };
};

const handleCommentProps = (
  config: any = {},
  basePath: string,
  labels: HomeLabels,
) => {
  return {
    title: config.title || labels.commentCard,
    items:
      config.list?.map((item: any) => ({
        ...item,
        avatar: getImagePath(item.avatar, basePath),
      })) || [],
  };
};

const handleBlockGridProps = (
  config: any = {},
  basePath: string,
  labels: HomeLabels,
) => {
  return {
    title: config.title || labels.blockGrid,
    basePath,
    items:
      config.list?.map((item: any) => ({
        ...item,
        url: getImagePath(item.url, basePath),
      })) || [],
  };
};

const handleQuestionProps = (config: any = {}, labels: HomeLabels) => {
  return {
    title: config.title || labels.faq,
    items: config.list || [],
  };
};

const componentMap = {
  banner: Banner,
  basic_doc: dynamic(() => import('@panda-wiki/ui').then(mod => mod.BasicDoc)),
  dir_doc: dynamic(() => import('@panda-wiki/ui').then(mod => mod.DirDoc)),
  simple_doc: dynamic(() =>
    import('@panda-wiki/ui').then(mod => mod.SimpleDoc),
  ),
  carousel: dynamic(() => import('@panda-wiki/ui').then(mod => mod.Carousel)),
  faq: dynamic(() => import('@panda-wiki/ui').then(mod => mod.Faq)),
  text: dynamic(() => import('@panda-wiki/ui').then(mod => mod.Text)),
  case: dynamic(() => import('@panda-wiki/ui').then(mod => mod.Case)),
  metrics: dynamic(() => import('@panda-wiki/ui').then(mod => mod.Metrics)),
  feature: dynamic(() => import('@panda-wiki/ui').then(mod => mod.Feature)),
  text_img: dynamic(() => import('@panda-wiki/ui').then(mod => mod.ImgText)),
  img_text: dynamic(() => import('@panda-wiki/ui').then(mod => mod.ImgText)),
  comment: dynamic(() => import('@panda-wiki/ui').then(mod => mod.Comment)),
  block_grid: dynamic(() =>
    import('@panda-wiki/ui').then(mod => mod.BlockGrid),
  ),
  question: dynamic(() => import('@panda-wiki/ui').then(mod => mod.Question)),
} as const;

type HomeLabels = {
  chatTab: string;
  viewMore: string;
  linkGroup: string;
  docSummaryCard: string;
  noSummary: string;
  docDirCard: string;
  simpleDocCard: string;
  carousel: string;
  title: string;
  case: string;
  metrics: string;
  feature: string;
  leftImgRightText: string;
  rightImgLeftText: string;
  commentCard: string;
  blockGrid: string;
  faq: string;
};

const Welcome = () => {
  const basePath = useBasePath();
  const { mobile = false, kbDetail, setQaModalOpen } = useStore();
  const { t } = useI18n();
  const settings = kbDetail?.settings;
  const labels: HomeLabels = {
    chatTab: t('qa.chatTab'),
    viewMore: t('common.viewMore'),
    linkGroup: t('home.linkGroup'),
    docSummaryCard: t('home.docSummaryCard'),
    noSummary: t('home.noSummary'),
    docDirCard: t('home.docDirCard'),
    simpleDocCard: t('home.simpleDocCard'),
    carousel: t('home.carousel'),
    title: t('home.title'),
    case: t('home.case'),
    metrics: t('home.metrics'),
    feature: t('home.feature'),
    leftImgRightText: t('home.leftImgRightText'),
    rightImgLeftText: t('home.rightImgLeftText'),
    commentCard: t('home.commentCard'),
    blockGrid: t('home.blockGrid'),
    faq: t('home.faq'),
  };
  const onBannerSearch = (
    searchText: string,
    type: 'chat' | 'search' = 'chat',
  ) => {
    if (searchText.trim()) {
      if (type === 'chat') {
        sessionStorage.setItem('chat_search_query', searchText.trim());
        setQaModalOpen?.(true);
      } else {
        sessionStorage.setItem('chat_search_query', searchText.trim());
      }
    }
  };

  const TYPE_TO_CONFIG_LABEL = {
    banner: 'banner_config',
    basic_doc: 'basic_doc_config',
    dir_doc: 'dir_doc_config',
    simple_doc: 'simple_doc_config',
    carousel: 'carousel_config',
    faq: 'faq_config',
    text: 'text_config',
    case: 'case_config',
    metrics: 'metrics_config',
    feature: 'feature_config',
    text_img: 'text_img_config',
    img_text: 'img_text_config',
    comment: 'comment_config',
    block_grid: 'block_grid_config',
    question: 'question_config',
  } as const;

  const handleComponentProps = (data: any) => {
    const config =
      data[
        TYPE_TO_CONFIG_LABEL[data.type as keyof typeof TYPE_TO_CONFIG_LABEL]
      ];

    switch (data.type) {
      case 'faq':
        return handleFaqProps(config, labels);
      case 'basic_doc':
        return handleBasicDocProps(config, data.nodes, basePath, labels);
      case 'dir_doc':
        return handleDirDocProps(config, data.nodes, basePath, labels);
      case 'simple_doc':
        return handleSimpleDocProps(config, data.nodes, basePath, labels);
      case 'carousel':
        return handleCarouselProps(config, basePath, labels);
      case 'banner':
        return {
          ...handleBannerProps(config, basePath, labels),
          onSearch: onBannerSearch,
          btns: (config?.btns || []).map((item: any) => ({
            ...item,
            href: getImagePath(item.href || '/node', basePath),
          })),
        };
      case 'text':
        return handleTextProps(config, labels);
      case 'case':
        return handleCaseProps(config, labels);
      case 'metrics':
        return handleMetricsProps(config, labels);
      case 'feature':
        return handleFeatureProps(config, labels);
      case 'text_img':
        return handleTextImgProps(config, basePath, labels);
      case 'img_text':
        return handleImgTextProps(config, basePath, labels);
      case 'comment':
        return handleCommentProps(config, basePath, labels);
      case 'block_grid':
        return handleBlockGridProps(config, basePath, labels);
      case 'question':
        return {
          ...handleQuestionProps(config, labels),
          onSearch: (text: string) => {
            onBannerSearch(text, 'chat');
          },
        };
    }
  };
  return (
    <>
      {settings?.web_app_landing_configs?.map((item, index) => {
        const Component = componentMap[item.type as keyof typeof componentMap];
        const props = handleComponentProps(item);
        return Component ? (
          // @ts-ignore
          <Component key={index} mobile={mobile} {...props} />
        ) : null;
      })}
    </>
  );
};

export default Welcome;
