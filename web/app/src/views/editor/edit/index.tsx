'use client';
import { getShareV1NodeDetail } from '@/request/ShareNode';
import { V1NodeDetailResp } from '@/request/types';
import { Box } from '@mui/material';
import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { useWrapContext } from '..';
import LoadingEditorWrap from './Loading';
import EditorWrap from './Wrap';

const Edit = () => {
  const { id = '' } = useParams();
  const { setNodeDetail, nodeDetail } = useWrapContext();
  // 有 id 时初始即为加载中，无 id 时直接进入编辑态，避免在 effect 内同步 setState
  const [loading, setLoading] = useState(() => Boolean(id));
  const [detail, setDetail] = useState<V1NodeDetailResp | null>(nodeDetail);

  const getDetail = useCallback(() => {
    // @ts-expect-error 类型错误
    getShareV1NodeDetail({
      id: id[0] as string,
    })
      .then(res => {
        setDetail(res as V1NodeDetailResp);
        setNodeDetail(res as V1NodeDetailResp);
        setTimeout(() => {
          window.scrollTo({ top: 0, behavior: 'smooth' });
        }, 0);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [id, setNodeDetail]);

  useEffect(() => {
    if (id) {
      getDetail();
    }
  }, [getDetail, id]);

  return (
    <Box
      sx={{
        position: 'relative',
        flexGrow: 1,
        /* Give a remote user a caret */
        '& .collaboration-carets__caret': {
          borderLeft: '1px solid #fff',
          borderRight: '1px solid #fff',
          marginLeft: '-1px',
          marginRight: '-1px',
          pointerEvents: 'none',
          position: 'relative',
          wordBreak: 'normal',
        },
        /* Render the username above the caret */
        '& .collaboration-carets__label': {
          borderRadius: '0 3px 3px 3px',
          color: '#fff',
          fontSize: '12px',
          fontStyle: 'normal',
          fontWeight: '600',
          left: '-1px',
          lineHeight: 'normal',
          padding: '0.1rem 0.3rem',
          position: 'absolute',
          top: '1.4em',
          userSelect: 'none',
          whiteSpace: 'nowrap',
        },
      }}
    >
      {loading ? <LoadingEditorWrap /> : <EditorWrap detail={detail!} />}
    </Box>
  );
};

export default Edit;
