/* eslint-disable @typescript-eslint/no-explicit-any */
import LottieComponent from 'lottie-react';
import { CSSProperties } from 'react';

const Lottie: any =
  (LottieComponent as any)?.default?.default ||
  (LottieComponent as any)?.default ||
  LottieComponent;

const LottieIcon = ({
  id,
  src,
  loop = true,
  autoplay = true,
  style,
}: {
  id: string;
  src: any;
  loop?: boolean;
  autoplay?: boolean;
  style?: CSSProperties;
}) => {
  if (!Lottie || typeof Lottie !== 'function') {
    return null;
  }
  return (
    <Lottie
      id={id}
      animationData={src}
      loop={loop}
      autoplay={autoplay}
      style={{ ...style }}
    />
  );
};

export default LottieIcon;
