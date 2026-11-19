import {
  type ActionType,
  PageContainer,
  type ProColumns,
  ProTable,
} from '@ant-design/pro-components';
import { FormattedMessage, useAccess, useIntl, useRequest } from '@umijs/max';
import { App, Descriptions, Drawer, Popconfirm, Table, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import { type Goods, type GoodsDetail, services } from '@/services';
import {
  GOODS_STATUS,
  optionalIntFilter,
  PAGE_SIZE,
  statusText,
  statusValueEnum,
  useCursorPager,
} from '../shared';

type GoodsQuery = {
  keyword?: string;
  status?: string;
};

const GoodsDrawer: React.FC<{ id?: number; onClose: () => void }> = ({
  id,
  onClose,
}) => {
  const intl = useIntl();
  const { canOperate } = useAccess();
  const { message } = App.useApp();
  const { data, loading, refresh } = useRequest(
    () => services.console.GetGoods({ id: id as number }),
    { ready: !!id, refreshDeps: [id], formatResult: (res: GoodsDetail) => res },
  );

  const changeSkuStatus = async (sku: { id?: number; status?: number }) => {
    if (sku.id === undefined) return;
    const status = sku.status === 1 ? 0 : 1;
    try {
      await services.console.SetGoodsSkuStatus({ id: sku.id, status });
      message.success(
        intl.formatMessage({ id: 'pages.planstar.goods.skuUpdateSuccess' }),
      );
      refresh();
    } catch {
      // 失败提示由 requestErrorConfig 统一处理。
    }
  };

  return (
    <Drawer
      open={!!id}
      onClose={onClose}
      width={640}
      title={intl.formatMessage(
        { id: 'pages.planstar.goods.detail.title' },
        { id },
      )}
      loading={loading}
    >
      <Descriptions column={2} size="small" bordered>
        <Descriptions.Item
          label={<FormattedMessage id="pages.planstar.goods.name" />}
          span={2}
        >
          {data?.goods?.name || '-'}
        </Descriptions.Item>
        <Descriptions.Item
          label={<FormattedMessage id="pages.planstar.goods.goodsSn" />}
          span={2}
        >
          {data?.goods?.goodsSn || '-'}
        </Descriptions.Item>
        <Descriptions.Item
          label={<FormattedMessage id="pages.planstar.goods.category" />}
        >
          {data?.goods?.category || '-'}
        </Descriptions.Item>
        <Descriptions.Item
          label={<FormattedMessage id="pages.planstar.goods.status" />}
        >
          {statusText(intl, GOODS_STATUS, data?.goods?.status)}
        </Descriptions.Item>
        <Descriptions.Item
          label={<FormattedMessage id="pages.planstar.goods.brief" />}
          span={2}
        >
          {data?.goods?.brief || '-'}
        </Descriptions.Item>
        <Descriptions.Item
          label={
            <FormattedMessage id="pages.planstar.goods.detail.description" />
          }
          span={2}
        >
          {data?.description || '-'}
        </Descriptions.Item>
      </Descriptions>

      <Typography.Title level={5} style={{ marginTop: 16 }}>
        <FormattedMessage id="pages.planstar.goods.detail.skus" />
      </Typography.Title>
      <Table
        rowKey="id"
        size="small"
        dataSource={data?.skus ?? []}
        pagination={false}
        columns={[
          {
            title: intl.formatMessage({
              id: 'pages.planstar.goods.detail.skuName',
            }),
            dataIndex: 'skuName',
          },
          {
            title: intl.formatMessage({
              id: 'pages.planstar.goods.detail.priceCoins',
            }),
            dataIndex: 'priceCoins',
            align: 'right' as const,
            render: (v: number) => v ?? 0,
          },
          {
            title: intl.formatMessage({
              id: 'pages.planstar.goods.detail.stock',
            }),
            dataIndex: 'stock',
            align: 'right' as const,
            render: (v: number) => v ?? 0,
          },
          {
            title: intl.formatMessage({
              id: 'pages.planstar.goods.detail.sold',
            }),
            dataIndex: 'sold',
            align: 'right' as const,
            render: (v: number) => v ?? 0,
          },
          {
            title: intl.formatMessage({ id: 'pages.planstar.goods.status' }),
            dataIndex: 'status',
            render: (v: number) => statusText(intl, GOODS_STATUS, v),
          },
          ...(canOperate
            ? [
                {
                  title: intl.formatMessage({ id: 'pages.planstar.operation' }),
                  dataIndex: 'option',
                  render: (
                    _: unknown,
                    sku: { id?: number; status?: number },
                  ) => {
                    const offing = sku.status === 1;
                    return (
                      <Popconfirm
                        title={intl.formatMessage({
                          id: offing
                            ? 'pages.planstar.goods.skuOffConfirm.title'
                            : 'pages.planstar.goods.skuOnConfirm.title',
                        })}
                        description={intl.formatMessage({
                          id: offing
                            ? 'pages.planstar.goods.skuOffConfirm.description'
                            : 'pages.planstar.goods.skuOnConfirm.description',
                        })}
                        onConfirm={() => changeSkuStatus(sku)}
                      >
                        <a>
                          {intl.formatMessage({
                            id: offing
                              ? 'pages.planstar.goods.off'
                              : 'pages.planstar.goods.on',
                          })}
                        </a>
                      </Popconfirm>
                    );
                  },
                },
              ]
            : []),
        ]}
      />
    </Drawer>
  );
};

const GoodsTable: React.FC = () => {
  const actionRef = useRef<ActionType>(null);
  const pager = useCursorPager();
  const { canOperate } = useAccess();
  const { message } = App.useApp();
  const intl = useIntl();
  const [activeId, setActiveId] = useState<number>();

  const changeStatus = async (record: Goods) => {
    const status = record.status === 1 ? 0 : 1;
    try {
      await services.console.SetGoodsStatus({ id: record.id, status });
      message.success(
        intl.formatMessage({ id: 'pages.planstar.goods.updateSuccess' }),
      );
      actionRef.current?.reload();
    } catch {
      // 失败提示由 requestErrorConfig 统一处理。
    }
  };

  const columns: ProColumns<Goods>[] = [
    {
      title: <FormattedMessage id="pages.planstar.search.keyword" />,
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: {
        placeholder: intl.formatMessage({
          id: 'pages.planstar.goods.keywordPlaceholder',
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
      title: <FormattedMessage id="pages.planstar.goods.goodsSn" />,
      dataIndex: 'goodsSn',
      search: false,
      width: 180,
      ellipsis: true,
      copyable: true,
    },
    {
      title: <FormattedMessage id="pages.planstar.goods.name" />,
      dataIndex: 'name',
      search: false,
      width: 220,
      ellipsis: true,
      render: (_, record) => (
        <Typography.Text strong>{record.name || '-'}</Typography.Text>
      ),
    },
    {
      title: <FormattedMessage id="pages.planstar.goods.category" />,
      dataIndex: 'category',
      search: false,
      width: 110,
      render: (_, record) => record.category || '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.goods.status" />,
      dataIndex: 'status',
      width: 110,
      valueEnum: statusValueEnum(intl, GOODS_STATUS),
    },
    {
      title: <FormattedMessage id="pages.planstar.goods.sort" />,
      dataIndex: 'sort',
      search: false,
      width: 90,
      align: 'right',
      render: (_, record) => record.sort ?? 0,
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
        const offing = record.status === 1;
        return [
          <a key="detail" onClick={() => setActiveId(record.id)}>
            <FormattedMessage id="pages.planstar.detail" />
          </a>,
          ...(canOperate
            ? [
                <Popconfirm
                  key="toggle"
                  title={intl.formatMessage({
                    id: offing
                      ? 'pages.planstar.goods.offConfirm.title'
                      : 'pages.planstar.goods.onConfirm.title',
                  })}
                  description={intl.formatMessage({
                    id: offing
                      ? 'pages.planstar.goods.offConfirm.description'
                      : 'pages.planstar.goods.onConfirm.description',
                  })}
                  onConfirm={() => changeStatus(record)}
                >
                  <a>
                    {intl.formatMessage({
                      id: offing
                        ? 'pages.planstar.goods.off'
                        : 'pages.planstar.goods.on',
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
      title={intl.formatMessage({ id: 'pages.planstar.goods.title' })}
      content={
        <Typography.Text type="secondary">
          <FormattedMessage id="pages.planstar.goods.description" />
        </Typography.Text>
      }
    >
      <ProTable<Goods, GoodsQuery>
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
          const res = await services.console.ListGoods({
            keyword,
            status: optionalIntFilter(status),
            pageSize: PAGE_SIZE,
            pageToken,
          });
          const data = res.goods ?? [];
          return {
            data,
            success: true,
            total: pager.total(current, data.length, res.nextPageToken),
          };
        }}
      />
      <GoodsDrawer id={activeId} onClose={() => setActiveId(undefined)} />
    </PageContainer>
  );
};

export default GoodsTable;
