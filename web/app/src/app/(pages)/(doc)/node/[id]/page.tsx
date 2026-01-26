import { getShareV1NodeDetail } from '@/request/ShareNode';
import { formatMeta } from '@/utils';
import Doc from '@/views/node';
import { ResolvingMetadata } from 'next';

export interface PageProps {
  params: Promise<{ id: string }>;
}

export async function generateMetadata(
  { params }: PageProps,
  parent: ResolvingMetadata,
) {
  const { id } = await params;
  let node = {
    name: 'Access denied',
    meta: {
      summary: 'Access denied',
    },
  };
  try {
    // @ts-ignore
    node = (await getShareV1NodeDetail({ id })) as any;
  } catch (error) {
    console.log(error);
  }

  return await formatMeta(
    { title: node?.name, description: node?.meta?.summary },
    parent,
  );
}

const DocPage = async ({ params }: PageProps) => {
  const { id = '' } = await params;
  let error: any = null;
  let node: any = null;
  try {
    // @ts-ignore
    node = await getShareV1NodeDetail({ id });
  } catch (err) {
    error = err;
  }
  return <Doc node={node} error={error} />;
};

export default DocPage;
