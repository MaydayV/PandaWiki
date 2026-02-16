'use client';
import { useStore } from '@/provider';
import { redirect } from 'next/navigation';
import React from 'react';

import { deepSearchFirstNode } from '@/utils';
import { useBasePath } from '@/hooks';

const NodePage = () => {
  const basePath = useBasePath();
  const { tree } = useStore();
  const firstNode = deepSearchFirstNode(tree || []);

  if (firstNode) {
    return redirect(`${basePath}/node/${firstNode.id}`);
  }

  return redirect(`${basePath}/home`);
};

export default NodePage;
