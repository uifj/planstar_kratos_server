import { type GoodsOrder, services } from '@/services';
import {
  PageContainer,
  type ActionType,
  type ProColumns,
  ProTable,
} from '@ant-design/pro-components';
import { FormattedMessage, useIntl } from '@umijs/max';
import { Typography } from 'antd';
import React, { useRef } from 'react';
import { PAGE_SIZE, useCursorPager } from '../shared';

type GoodsOrderQuery = {
  keyword?: string;
  status?: string;
};

const GoodsOrderTable: React.FC = () => {
  const actionRef = useRef<ActionType>(null);
  const pager = useCursorPager();
  const intl = useIntl();

  const columns: ProColumns<GoodsOrder>[] = [
    {
      title: <FormattedMessage id="pages.planstar.search.keyword" />,
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: {
        placeholder: intl.formatMessage({
          id: 'pages.planstar.goodsOrders.keywordPlaceholder',
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
      title: <FormattedMessage id="pages.planstar.goodsOrders.orderNo" />,
      dataIndex: 'orderNo',
      search: false,
      width: 200,
      ellipsis: true,
      copyable: true,
    },
    {
      title: <FormattedMessage id="pages.planstar.userId" />,
      dataIndex: 'userId',
      search: false,
      width: 110,
      render: (_, record) => record.userId ?? '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.goodsOrders.skuName" />,
      dataIndex: 'skuName',
      search: false,
      width: 180,
      ellipsis: true,
      render: (_, record) => record.skuName || '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.goodsOrders.priceCoins" />,
      dataIndex: 'priceCoins',
      search: false,
      width: 110,
      align: 'right',
      render: (_, record) => record.priceCoins ?? 0,
    },
    {
      title: <FormattedMessage id="pages.planstar.goodsOrders.status" />,
      dataIndex: 'status',
      width: 120,
      valueEnum: {
        paid: { text: intl.formatMessage({ id: 'pages.planstar.goodsOrders.status.paid' }), status: 'Processing' },
        completed: { text: intl.formatMessage({ id: 'pages.planstar.goodsOrders.status.completed' }), status: 'Success' },
        cancelled: { text: intl.formatMessage({ id: 'pages.planstar.goodsOrders.status.cancelled' }), status: 'Default' },
      },
    },
    {
      title: <FormattedMessage id="pages.planstar.createdAt" />,
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 200,
      search: false,
    },
  ];

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'pages.planstar.goodsOrders.title' })}
      content={
        <Typography.Text type="secondary">
          <FormattedMessage id="pages.planstar.goodsOrders.description" />
        </Typography.Text>
      }
    >
      <ProTable<GoodsOrder, GoodsOrderQuery>
        actionRef={actionRef}
        rowKey="id"
        cardBordered
        columns={columns}
        search={{ labelWidth: 'auto' }}
        options={{ density: false, fullScreen: false, setting: false }}
        pagination={{ pageSize: PAGE_SIZE, showSizeChanger: false }}
        request={async (params) => {
          const keyword = params.keyword?.trim() || undefined;
          const status = params.status || undefined;
          const current = params.current ?? 1;
          const pageToken = pager.pageToken(
            current,
            `${keyword ?? ''}|${status ?? ''}`,
          );
          const res = await services.console.ListGoodsOrders({
            keyword,
            status,
            pageSize: PAGE_SIZE,
            pageToken,
          });
          const data = res.orders ?? [];
          return {
            data,
            success: true,
            total: pager.total(current, data.length, res.nextPageToken),
          };
        }}
      />
    </PageContainer>
  );
};

export default GoodsOrderTable;