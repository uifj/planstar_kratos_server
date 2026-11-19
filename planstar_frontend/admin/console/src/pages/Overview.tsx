import { TrendChart } from '@/components';
import { type Dashboard, services } from '@/services';
import { PageContainer } from '@ant-design/pro-components';
import { FormattedMessage, Link, useRequest } from '@umijs/max';
import {
  Alert,
  Card,
  Col,
  Progress,
  Row,
  Statistic,
  Typography,
  theme,
} from 'antd';
import React from 'react';
import { formatCents } from './planstar/shared';

const PUBLISH_KEYS: Record<number, string> = {
  0: 'pages.planstar.projects.publish.draft',
  1: 'pages.planstar.projects.publish.published',
  2: 'pages.planstar.projects.publish.reviewing',
  3: 'pages.planstar.projects.publish.rejected',
};

const TREND_KEYS: Record<string, string> = {
  orders: 'pages.overview.trend.orders',
  pledges: 'pages.overview.trend.pledges',
  votes: 'pages.overview.trend.votes',
  new_users: 'pages.overview.trend.new_users',
};

const Overview: React.FC = () => {
  const { token } = theme.useToken();

  // GetDashboard 直接返回 Dashboard，用 formatResult 原样透传。
  const { data, loading, error } = useRequest(
    () => services.console.GetDashboard({}),
    { formatResult: (res: Dashboard) => res },
  );

  // 请求失败时显示 "-" 而不是 0，避免把「加载失败」说成「一条都没有」。
  const stat = (value?: number) => (error ? '-' : (value ?? 0));
  const money = (cents?: number) => (error ? '-' : formatCents(cents));

  const votesTotal = data?.votesTotal ?? 0;
  const supportPercent =
    votesTotal > 0
      ? Math.round(((data?.votesSupport ?? 0) / votesTotal) * 100)
      : 0;
  const projectsTotal = data?.projectsTotal ?? 0;
  const buckets = data?.projectsByPublishStatus ?? [];

  return (
    <PageContainer
      title={<FormattedMessage id="pages.overview.title" />}
      content={
        <Typography.Text type="secondary">
          <FormattedMessage id="pages.overview.description" />
        </Typography.Text>
      }
    >
      {error && (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={<FormattedMessage id="pages.overview.loadFailed" />}
        />
      )}

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} xl={6}>
          <Card loading={loading}>
            <Statistic
              title={<FormattedMessage id="pages.overview.users" />}
              value={stat(data?.usersTotal)}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card loading={loading}>
            <Statistic
              title={<FormattedMessage id="pages.overview.projects" />}
              value={stat(data?.projectsTotal)}
              suffix={
                <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                  <FormattedMessage
                    id="pages.overview.projectsOngoing"
                    values={{ count: stat(data?.projectsOngoing) }}
                  />
                </Typography.Text>
              }
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card loading={loading}>
            <Statistic
              title={<FormattedMessage id="pages.overview.raised" />}
              value={money(data?.raisedTotal)}
              styles={{ content: { color: token.colorSuccess } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card loading={loading}>
            <Statistic
              title={<FormattedMessage id="pages.overview.gmv" />}
              value={money(data?.gmv)}
              suffix={
                <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                  <FormattedMessage
                    id="pages.overview.ordersPaid"
                    values={{ count: stat(data?.ordersPaid) }}
                  />
                </Typography.Text>
              }
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} sm={12} xl={6}>
          <Card loading={loading}>
            <Statistic
              title={<FormattedMessage id="pages.overview.ideas" />}
              value={`${stat(data?.ideasPublished)} / ${stat(data?.ideasTotal)}`}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card loading={loading}>
            <Statistic
              title={<FormattedMessage id="pages.overview.votes" />}
              value={stat(data?.votesTotal)}
            />
            {/* 没有投票时不画比例条：整条 trail 会被误读成「全部反对」。 */}
            {votesTotal > 0 && (
              <Progress
                percent={supportPercent}
                strokeColor={token.colorSuccess}
                trailColor={token.colorError}
                showInfo={false}
              />
            )}
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              <FormattedMessage
                id="pages.overview.votesSplit"
                values={{
                  support: stat(data?.votesSupport),
                  oppose: stat(data?.votesOppose),
                }}
              />
            </Typography.Text>
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card loading={loading}>
            <Statistic
              title={<FormattedMessage id="pages.overview.pledges" />}
              value={stat(data?.pledgesPaid)}
              suffix={
                <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                  {money(data?.pledgedAmount)}
                </Typography.Text>
              }
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card loading={loading}>
            <Statistic
              title={<FormattedMessage id="pages.overview.coins" />}
              value={stat(data?.coinsSpent)}
              suffix={
                <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                  <FormattedMessage
                    id="pages.overview.goodsOrders"
                    values={{ count: stat(data?.goodsOrdersPaid) }}
                  />
                </Typography.Text>
              }
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} xl={12}>
          <Card
            loading={loading}
            title={<FormattedMessage id="pages.overview.pending" />}
          >
            <Row gutter={[16, 16]}>
              <Col span={8}>
                <Statistic
                  title={
                    <FormattedMessage id="pages.overview.refundsPending" />
                  }
                  value={stat(data?.refundsPending)}
                  styles={
                    (data?.refundsPending ?? 0) > 0
                      ? { content: { color: token.colorWarning } }
                      : undefined
                  }
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title={
                    <FormattedMessage id="pages.overview.reportsPending" />
                  }
                  value={stat(data?.reportsPending)}
                  styles={
                    (data?.reportsPending ?? 0) > 0
                      ? { content: { color: token.colorWarning } }
                      : undefined
                  }
                />
                <Link to="/planstar/reports">
                  <FormattedMessage id="pages.overview.viewReports" />
                </Link>
              </Col>
              <Col span={8}>
                <Statistic
                  title={<FormattedMessage id="pages.overview.feedsHidden" />}
                  value={stat(data?.feedsHidden)}
                />
              </Col>
            </Row>
          </Card>
        </Col>
        <Col xs={24} xl={12}>
          <Card
            loading={loading}
            title={<FormattedMessage id="pages.overview.publishDist" />}
          >
            {buckets.length === 0 ? (
              <Typography.Text type="secondary">-</Typography.Text>
            ) : (
              buckets.map((bucket) => {
                const count = bucket.count ?? 0;
                const percent =
                  projectsTotal > 0
                    ? Math.round((count / projectsTotal) * 100)
                    : 0;
                return (
                  <div key={bucket.status} style={{ marginBottom: 8 }}>
                    <Typography.Text style={{ fontSize: 13 }}>
                      <FormattedMessage
                        id={
                          PUBLISH_KEYS[bucket.status ?? -1] ??
                          'pages.overview.publishDist'
                        }
                      />
                      {` · ${count}`}
                    </Typography.Text>
                    <Progress percent={percent} size="small" />
                  </div>
                );
              })
            )}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        {(data?.trends ?? []).map((series) => (
          <Col key={series.name} xs={24} sm={12} xl={6}>
            <Card
              loading={loading}
              title={
                <FormattedMessage
                  id={TREND_KEYS[series.name ?? ''] ?? 'pages.overview.trends'}
                />
              }
            >
              <TrendChart
                points={series.points ?? []}
                color={token.colorPrimary}
              />
            </Card>
          </Col>
        ))}
      </Row>
    </PageContainer>
  );
};

export default Overview;
