import { type Idea, type IdeaDetail, services } from '@/services';
import {
  PageContainer,
  type ActionType,
  type ProColumns,
  ProTable,
} from '@ant-design/pro-components';
import { FormattedMessage, useIntl, useRequest } from '@umijs/max';
import {
  Descriptions,
  Drawer,
  Progress,
  Table,
  Tag,
  Typography,
  theme,
} from 'antd';
import React, { useRef, useState } from 'react';
import {
  IDEA_PUBLISH_STATUS,
  PAGE_SIZE,
  optionalIntFilter,
  statusText,
  statusValueEnum,
  useCursorPager,
} from '../shared';

type IdeaQuery = {
  keyword?: string;
  publishStatus?: string;
};

const IdeaDrawer: React.FC<{ id?: number; onClose: () => void }> = ({
  id,
  onClose,
}) => {
  const intl = useIntl();
  const { token } = theme.useToken();
  const { data, loading } = useRequest(
    () => services.console.GetIdea({ id: id as number }),
    { ready: !!id, refreshDeps: [id], formatResult: (res: IdeaDetail) => res },
  );

  const votesTotal = data?.votesTotal ?? 0;
  const supportPercent =
    votesTotal > 0
      ? Math.round(((data?.votesSupport ?? 0) / votesTotal) * 100)
      : 0;

  return (
    <Drawer
      open={!!id}
      onClose={onClose}
      width={640}
      title={intl.formatMessage(
        { id: 'pages.planstar.ideas.detail.title' },
        { id },
      )}
      loading={loading}
    >
      <Descriptions column={2} size="small" bordered>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.ideas.titleColumn" />} span={2}>
          {data?.idea?.title || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.ideas.category" />}>
          {data?.idea?.categoryName || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.ideas.publishStatus" />}>
          {statusText(intl, IDEA_PUBLISH_STATUS, data?.idea?.publishStatus)}
        </Descriptions.Item>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.ideas.support" />}>
          {data?.idea?.supportCount ?? 0}
        </Descriptions.Item>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.ideas.oppose" />}>
          {data?.idea?.opposeCount ?? 0}
        </Descriptions.Item>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.ideas.coinIntent" />} span={2}>
          {data?.idea?.coinIntentSum ?? 0}
        </Descriptions.Item>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.ideas.detail.description" />} span={2}>
          {data?.description || '-'}
        </Descriptions.Item>
      </Descriptions>

      <Typography.Title level={5} style={{ marginTop: 16 }}>
        <FormattedMessage id="pages.planstar.ideas.detail.votes" />
      </Typography.Title>
      <Typography.Text type="secondary">
        <FormattedMessage
          id="pages.planstar.ideas.detail.votesSplit"
          values={{
            total: votesTotal,
            support: data?.votesSupport ?? 0,
            oppose: data?.votesOppose ?? 0,
          }}
        />
      </Typography.Text>
      {votesTotal > 0 && (
        <Progress
          percent={supportPercent}
          strokeColor={token.colorSuccess}
          trailColor={token.colorError}
          showInfo={false}
        />
      )}

      <Typography.Title level={5} style={{ marginTop: 16 }}>
        <FormattedMessage id="pages.planstar.ideas.detail.recentVotes" />
      </Typography.Title>
      <Table
        rowKey="id"
        size="small"
        dataSource={data?.recentVotes ?? []}
        pagination={false}
        columns={[
          {
            title: intl.formatMessage({ id: 'pages.planstar.ideas.detail.user' }),
            dataIndex: 'userId',
          },
          {
            title: intl.formatMessage({ id: 'pages.planstar.ideas.detail.voteType' }),
            dataIndex: 'voteType',
            render: (v: number) =>
              v === 1 ? (
                <Tag color="success">
                  {intl.formatMessage({ id: 'pages.planstar.ideas.detail.vote.support' })}
                </Tag>
              ) : (
                <Tag color="error">
                  {intl.formatMessage({ id: 'pages.planstar.ideas.detail.vote.oppose' })}
                </Tag>
              ),
          },
          {
            title: intl.formatMessage({ id: 'pages.planstar.ideas.detail.coinTier' }),
            dataIndex: 'coinTier',
            render: (v: number) => v ?? 0,
          },
          {
            title: intl.formatMessage({ id: 'pages.planstar.ideas.detail.reason' }),
            dataIndex: 'reasonText',
            render: (v: string) => v || '-',
          },
        ]}
      />
    </Drawer>
  );
};

const IdeaTable: React.FC = () => {
  const actionRef = useRef<ActionType>(null);
  const pager = useCursorPager();
  const intl = useIntl();
  const [activeId, setActiveId] = useState<number>();

  const columns: ProColumns<Idea>[] = [
    {
      title: <FormattedMessage id="pages.planstar.search.keyword" />,
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: {
        placeholder: intl.formatMessage({
          id: 'pages.planstar.ideas.keywordPlaceholder',
        }),
      },
    },
    {
      title: 'ID',
      dataIndex: 'id',
      width: 90,
      search: false,
      copyable: true,
    },
    {
      title: <FormattedMessage id="pages.planstar.ideas.titleColumn" />,
      dataIndex: 'title',
      search: false,
      width: 220,
      ellipsis: true,
      render: (_, record) => (
        <Typography.Text strong>{record.title || '-'}</Typography.Text>
      ),
    },
    {
      title: <FormattedMessage id="pages.planstar.ideas.category" />,
      dataIndex: 'categoryName',
      search: false,
      width: 120,
      render: (_, record) => record.categoryName || '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.ideas.publishStatus" />,
      dataIndex: 'publishStatus',
      width: 110,
      valueEnum: statusValueEnum(intl, IDEA_PUBLISH_STATUS),
    },
    {
      title: <FormattedMessage id="pages.planstar.ideas.support" />,
      dataIndex: 'supportCount',
      search: false,
      width: 90,
      align: 'right',
      render: (_, record) => record.supportCount ?? 0,
    },
    {
      title: <FormattedMessage id="pages.planstar.ideas.oppose" />,
      dataIndex: 'opposeCount',
      search: false,
      width: 90,
      align: 'right',
      render: (_, record) => record.opposeCount ?? 0,
    },
    {
      title: <FormattedMessage id="pages.planstar.ideas.coinIntent" />,
      dataIndex: 'coinIntentSum',
      search: false,
      width: 110,
      align: 'right',
      render: (_, record) => record.coinIntentSum ?? 0,
    },
    {
      title: <FormattedMessage id="pages.planstar.createdAt" />,
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 200,
      search: false,
    },
    {
      title: <FormattedMessage id="pages.planstar.operation" />,
      dataIndex: 'option',
      valueType: 'option',
      width: 80,
      search: false,
      render: (_, record) => [
        <a key="detail" onClick={() => setActiveId(record.id)}>
          <FormattedMessage id="pages.planstar.detail" />
        </a>,
      ],
    },
  ];

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'pages.planstar.ideas.title' })}
      content={
        <Typography.Text type="secondary">
          <FormattedMessage id="pages.planstar.ideas.description" />
        </Typography.Text>
      }
    >
      <ProTable<Idea, IdeaQuery>
        actionRef={actionRef}
        rowKey="id"
        cardBordered
        columns={columns}
        search={{ labelWidth: 'auto' }}
        options={{ density: false, fullScreen: false, setting: false }}
        pagination={{ pageSize: PAGE_SIZE, showSizeChanger: false }}
        request={async (params) => {
          const keyword = params.keyword?.trim() || undefined;
          const publishStatus =
            params.publishStatus === undefined || params.publishStatus === ''
              ? undefined
              : Number(params.publishStatus);
          const current = params.current ?? 1;
          const pageToken = pager.pageToken(
            current,
            `${keyword ?? ''}|${publishStatus ?? ''}`,
          );
          const res = await services.console.ListIdeas({
            keyword,
            publishStatus: optionalIntFilter(publishStatus),
            pageSize: PAGE_SIZE,
            pageToken,
          });
          const data = res.ideas ?? [];
          return {
            data,
            success: true,
            total: pager.total(current, data.length, res.nextPageToken),
          };
        }}
      />
      <IdeaDrawer id={activeId} onClose={() => setActiveId(undefined)} />
    </PageContainer>
  );
};

export default IdeaTable;