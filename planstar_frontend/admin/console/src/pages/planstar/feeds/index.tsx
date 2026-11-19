import {
  type ActionType,
  PageContainer,
  type ProColumns,
  ProTable,
} from '@ant-design/pro-components';
import { FormattedMessage, useAccess, useIntl, useRequest } from '@umijs/max';
import {
  App,
  Descriptions,
  Drawer,
  Image,
  Popconfirm,
  Space,
  Tag,
  Typography,
} from 'antd';
import React, { useRef, useState } from 'react';
import { type Feed, type FeedDetail, services } from '@/services';
import {
  FEED_STATUS,
  optionalIntFilter,
  PAGE_SIZE,
  statusText,
  statusValueEnum,
  useCursorPager,
} from '../shared';

type FeedQuery = {
  keyword?: string;
  status?: string;
};

const FeedDrawer: React.FC<{ id?: number; onClose: () => void }> = ({
  id,
  onClose,
}) => {
  const intl = useIntl();
  const { data, loading } = useRequest(
    () => services.console.GetFeed({ id: id as number }),
    { ready: !!id, refreshDeps: [id], formatResult: (res: FeedDetail) => res },
  );

  return (
    <Drawer
      open={!!id}
      onClose={onClose}
      width={640}
      title={intl.formatMessage(
        { id: 'pages.planstar.feeds.detail.title' },
        { id },
      )}
      loading={loading}
    >
      <Descriptions column={2} size="small" bordered>
        <Descriptions.Item
          label={<FormattedMessage id="pages.planstar.feeds.titleColumn" />}
          span={2}
        >
          {data?.feed?.title || '-'}
        </Descriptions.Item>
        <Descriptions.Item
          label={<FormattedMessage id="pages.planstar.feeds.detail.author" />}
        >
          {data?.authorName || '-'}
        </Descriptions.Item>
        <Descriptions.Item
          label={<FormattedMessage id="pages.planstar.userId" />}
        >
          {data?.feed?.userId ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item
          label={<FormattedMessage id="pages.planstar.feeds.likes" />}
        >
          {data?.feed?.likes ?? 0}
        </Descriptions.Item>
        <Descriptions.Item
          label={<FormattedMessage id="pages.planstar.feeds.replies" />}
        >
          {data?.feed?.replyCount ?? 0}
        </Descriptions.Item>
        <Descriptions.Item
          label={<FormattedMessage id="pages.planstar.feeds.views" />}
        >
          {data?.feed?.views ?? 0}
        </Descriptions.Item>
        <Descriptions.Item
          label={<FormattedMessage id="pages.planstar.feeds.status" />}
        >
          {statusText(intl, FEED_STATUS, data?.feed?.status)}
        </Descriptions.Item>
        <Descriptions.Item
          label={<FormattedMessage id="pages.planstar.feeds.detail.content" />}
          span={2}
        >
          {data?.feed?.content || '-'}
        </Descriptions.Item>
        <Descriptions.Item
          label={<FormattedMessage id="pages.planstar.feeds.detail.tags" />}
          span={2}
        >
          <Space wrap>
            {(data?.tags ?? []).map((tag) => (
              <Tag key={tag}>{tag}</Tag>
            ))}
            {(data?.tags ?? []).length === 0 && '-'}
          </Space>
        </Descriptions.Item>
      </Descriptions>

      {(data?.images ?? []).length > 0 && (
        <>
          <Typography.Title level={5} style={{ marginTop: 16 }}>
            <FormattedMessage id="pages.planstar.feeds.detail.images" />
          </Typography.Title>
          <Space wrap>
            {(data?.images ?? []).map((src) => (
              <Image key={src} src={src} width={120} />
            ))}
          </Space>
        </>
      )}
    </Drawer>
  );
};

const FeedTable: React.FC = () => {
  const actionRef = useRef<ActionType>(null);
  const pager = useCursorPager();
  const { canOperate } = useAccess();
  const { message } = App.useApp();
  const intl = useIntl();
  const [activeId, setActiveId] = useState<number>();

  const changeStatus = async (record: Feed) => {
    const status = record.status === 1 ? 0 : 1;
    try {
      await services.console.SetFeedStatus({ id: record.id, status });
      message.success(
        intl.formatMessage({ id: 'pages.planstar.feeds.updateSuccess' }),
      );
      actionRef.current?.reload();
    } catch {
      // 失败提示由 requestErrorConfig 统一处理。
    }
  };

  const columns: ProColumns<Feed>[] = [
    {
      title: <FormattedMessage id="pages.planstar.search.keyword" />,
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: {
        placeholder: intl.formatMessage({
          id: 'pages.planstar.feeds.keywordPlaceholder',
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
      title: <FormattedMessage id="pages.planstar.feeds.titleColumn" />,
      dataIndex: 'title',
      search: false,
      width: 220,
      ellipsis: true,
      render: (_, record) => (
        <Typography.Text strong>{record.title || '-'}</Typography.Text>
      ),
    },
    {
      title: <FormattedMessage id="pages.planstar.userId" />,
      dataIndex: 'userId',
      search: false,
      width: 110,
      render: (_, record) => record.userId ?? '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.feeds.likes" />,
      dataIndex: 'likes',
      search: false,
      width: 90,
      align: 'right',
      render: (_, record) => record.likes ?? 0,
    },
    {
      title: <FormattedMessage id="pages.planstar.feeds.replies" />,
      dataIndex: 'replyCount',
      search: false,
      width: 90,
      align: 'right',
      render: (_, record) => record.replyCount ?? 0,
    },
    {
      title: <FormattedMessage id="pages.planstar.feeds.views" />,
      dataIndex: 'views',
      search: false,
      width: 100,
      align: 'right',
      render: (_, record) => record.views ?? 0,
    },
    {
      title: <FormattedMessage id="pages.planstar.feeds.status" />,
      dataIndex: 'status',
      width: 110,
      valueEnum: statusValueEnum(intl, FEED_STATUS),
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
      width: 140,
      search: false,
      render: (_, record) => {
        const hiding = record.status === 1;
        return [
          <a key="detail" onClick={() => setActiveId(record.id)}>
            <FormattedMessage id="pages.planstar.detail" />
          </a>,
          ...(canOperate
            ? [
                <Popconfirm
                  key="toggle"
                  title={intl.formatMessage({
                    id: hiding
                      ? 'pages.planstar.feeds.hideConfirm.title'
                      : 'pages.planstar.feeds.showConfirm.title',
                  })}
                  description={intl.formatMessage({
                    id: hiding
                      ? 'pages.planstar.feeds.hideConfirm.description'
                      : 'pages.planstar.feeds.showConfirm.description',
                  })}
                  onConfirm={() => changeStatus(record)}
                >
                  <a>
                    {intl.formatMessage({
                      id: hiding
                        ? 'pages.planstar.feeds.hide'
                        : 'pages.planstar.feeds.show',
                    })}
                  </a>
                </Popconfirm>,
              ]
            : []),
        ];
      },
    },
  ];

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'pages.planstar.feeds.title' })}
      content={
        <Typography.Text type="secondary">
          <FormattedMessage id="pages.planstar.feeds.description" />
        </Typography.Text>
      }
    >
      <ProTable<Feed, FeedQuery>
        actionRef={actionRef}
        rowKey="id"
        cardBordered
        columns={columns}
        search={{ labelWidth: 'auto' }}
        options={{ density: false, fullScreen: false, setting: false }}
        pagination={{ pageSize: PAGE_SIZE, showSizeChanger: false }}
        request={async (params) => {
          const keyword = params.keyword?.trim() || undefined;
          const status =
            params.status === undefined || params.status === ''
              ? undefined
              : Number(params.status);
          const current = params.current ?? 1;
          const pageToken = pager.pageToken(
            current,
            `${keyword ?? ''}|${status ?? ''}`,
          );
          const res = await services.console.ListFeeds({
            keyword,
            status: optionalIntFilter(status),
            pageSize: PAGE_SIZE,
            pageToken,
          });
          const data = res.feeds ?? [];
          return {
            data,
            success: true,
            total: pager.total(current, data.length, res.nextPageToken),
          };
        }}
      />
      <FeedDrawer id={activeId} onClose={() => setActiveId(undefined)} />
    </PageContainer>
  );
};

export default FeedTable;
