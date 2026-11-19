/**
 * Planstar Kratos 服务端接口类型。
 *
 * 全部字段与 `planstar_kratos_server/planstar/internal/biz` 的 JSON tag 对齐；
 * 金额一律为**分**（服务端把数据库里的元 ×100 后下发）。
 */

export type FundingStatus = 'preheating' | 'ongoing' | 'achieved' | 'delivered' | 'failed';

export type RankingType = 'hot' | 'achievement' | 'ending';

export type ProjectSort = 'hot' | 'new' | 'ending' | 'achievement';

export interface Banner {
  id: number | string;
  title: string;
  image: string;
  url?: string;
  showId?: string | number;
}

export interface SubCategoryTag {
  subCategoryId: number;
  categoryId: number;
  name: string;
}

export interface Category {
  id: number;
  name: string;
  enName: string;
  icon?: string;
  sort: number;
  isSelect: boolean;
  status: number;
  subCategoryTags: SubCategoryTag[];
}

/** 企划卡片（`/v1/projects`、`/v1/projects/hot`、榜单内嵌）。 */
export interface ProjectSummary {
  projectId: number;
  projectNo: string;
  title: string;
  subtitle?: string;
  cover: string;
  categoryId: number;
  categoryName: string;
  projectType: string;
  creatorId: number;
  creatorName: string;
  /** 目标金额（分）。 */
  goalAmount: number;
  /** 已筹金额（分）。 */
  raisedAmount: number;
  supporterCount: number;
  /** 截止时间（Unix 毫秒）。 */
  deadline: number;
  remainingDays: number;
  fundingStatus: FundingStatus;
  /** 达成率，千分比（860 表示 86%），展示前用 `permilleToPercent` 转换。 */
  achievementRate: number;
  /** 最低支持金额（分）。 */
  minSupportAmount: number;
  tags: string[];
  isFeatured: boolean;
}

export interface CreatorProfile {
  creatorId: number;
  displayName: string;
  avatar: string;
  isVerified: boolean;
  projectNum: number;
  successNum: number;
  /** 履约率，千分比（990 表示 99%）。 */
  fulfillmentRate: number;
  bio?: string;
}

export interface RewardTier {
  rewardTierId: number;
  title: string;
  /** 支持金额（分）。 */
  price: number;
  rewardType: string;
  items: string[];
  limitNum: number;
  soldNum: number;
  /** 不限量档位为 undefined。 */
  remainingNum?: number;
  estimatedDeliveryTime?: string;
  isEarlyBird: boolean;
  isLimited: boolean;
}

export interface ProjectFaq {
  faqId: number;
  question: string;
  answer: string;
  sort: number;
}

export interface ProjectUpdate {
  updateId: number;
  title: string;
  content: string;
  images: string[];
  isSupporterOnly: boolean;
  /** 发布时间（Unix 毫秒）。 */
  publishTime: number;
}

export interface StretchGoal {
  goalId: number;
  title: string;
  description?: string;
  targetAmount: number;
  currentAmount: number;
  isAchieved: boolean;
  sort: number;
}

export interface ProjectDetail extends ProjectSummary {
  story: string;
  gallery: string[];
  riskNote: string;
  deliveryPlan: string;
  fundUsage: string;
  faq: ProjectFaq[];
  refundRule: string;
  refundDays: number;
  ratingAvg: number;
  reviewCount: number;
  showCount: number;
  creator: CreatorProfile | null;
  rewardTiers: RewardTier[];
  updates: ProjectUpdate[];
  stretchGoals: StretchGoal[];
}

export interface ProjectReview {
  reviewId: number;
  projectId: number;
  userName: string;
  userAvatar?: string;
  rating: number;
  content?: string;
  images: string[];
  isShow: boolean;
  isAnonymous: boolean;
  likeCount: number;
  /** 创建时间（Unix 毫秒）。 */
  gmtCreate: number;
}

export interface ReviewPage {
  items: ProjectReview[];
  ratingAvg: number;
  reviewCount: number;
  showCount: number;
}

export interface RankingItem {
  rank: number;
  project: ProjectSummary;
  /** 榜单指标值：热度榜为热度值，达成榜为达成率（千分比），冲刺榜为剩余天数。 */
  metricValue: number;
}

export interface ProjectRanking {
  type: string;
  /** 生成时间（Unix 毫秒）。 */
  generatedAt: number;
  items: RankingItem[];
}

export interface PageParams {
  page?: number;
  pageSize?: number;
}

export interface ProjectListParams extends PageParams {
  categoryId?: number;
  fundingStatus?: FundingStatus;
  sort?: ProjectSort;
}

export interface RankingParams {
  type?: RankingType;
  categoryId?: number;
  /** 服务端上限 50。 */
  limit?: number;
}

export interface ReviewParams extends PageParams {
  sort?: 'latest' | 'hot';
  onlyShow?: boolean;
}
