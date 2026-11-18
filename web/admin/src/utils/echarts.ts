/**
 * echarts 按需注册入口。
 *
 * 全站图表组件(PieTrend / BarTrend / MapChart)统一从这里引入 echarts,
 * 仅注册实际用到的图表类型、组件与渲染器,避免全量引入 echarts 主包。
 * 新增图表类型时,需要在这里补充对应注册项。
 */
import { BarChart, MapChart, PieChart } from 'echarts/charts';
import {
  GridComponent,
  TooltipComponent,
  VisualMapComponent,
} from 'echarts/components';
import * as echarts from 'echarts/core';
import { CanvasRenderer, SVGRenderer } from 'echarts/renderers';

echarts.use([
  BarChart,
  PieChart,
  MapChart,
  GridComponent,
  TooltipComponent,
  VisualMapComponent,
  CanvasRenderer,
  SVGRenderer,
]);

export default echarts;
