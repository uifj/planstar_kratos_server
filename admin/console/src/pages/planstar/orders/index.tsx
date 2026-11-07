import { type Order, services } from '@/services';
import {
  PageContainer,
  type ActionType,
  type ProColumns,
  ProTable,
} from '@ant-design/pro-components';
import { FormattedMessage, useIntl } from '@umijs/max';
import { Typography } from 'antd';
import React, { useRef } from 'react';
import {
  PAGE_SIZE,
  formatCents,
  optionalIntFilter,
  useCursorPager,
} from '../shared';

type OrderQuery = {
  keyword?: string;
  status?: string;
};

const OrderTable: React.FC = () => {
  const actionRef = useRef<ActionType>(null);
  const pager = useCursorPager();
  const intl = useIntl();

  const columns: ProColumns<Order>[] = [
    {
      title: <FormattedMessage id="pages.planstar.search.keyword" />,
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: {
        placeholder: intl.formatMessage({
          id: 'pages.planstar.orders.keywordPlaceholder',
        }),
      },
    },
    {
      title: <FormattedMessage id="pages.planstar.orders.orderNo" />,
      dataIndex: 'orderNo',
      copyable: true,
      ellipsis: true,
      search: false,
      render: (_, record) => record.orderNo || '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.orders.userId" />,
      dataIndex: 'userId',
      width: 110,
      search: false,
      render: (_, record) => record.userId ?? '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.orders.amount" />,
      dataIndex: 'amount',
      width: 130,
      align: 'right',
      search: false,
      render: (_, record) => formatCents(record.amount),
    },
    {
      title: <FormattedMessage id="pages.planstar.orders.status" />,
      dataIndex: 'status',
      width: 120,
      valueEnum: {
        1: {
          text: <FormattedMessage id="pages.planstar.orders.status.init" />,
          status: 'Default',
        },
        2: {
          text: <FormattedMessage id="pages.planstar.orders.status.paying" />,
          status: 'Processing',
        },
        3: {
          text: (
            <FormattedMessage id="pages.planstar.orders.status.waitingForTickets" />
          ),
          status: 'Processing',
        },
        4: {
          text: (
            <FormattedMessage id="pages.planstar.orders.status.gettingTickets" />
          ),
          status: 'Processing',
        },
        5: {
          text: <FormattedMessage id="pages.planstar.orders.status.finished" />,
          status: 'Success',
        },
        6: {
          text: <FormattedMessage id="pages.planstar.orders.status.canceled" />,
          status: 'Error',
        },
        7: {
          text: <FormattedMessage id="pages.planstar.orders.status.refunded" />,
          status: 'Default',
        },
      },
    },
    {
      title: <FormattedMessage id="pages.planstar.orders.createdAt" />,
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 200,
      search: false,
    },
  ];

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'pages.planstar.orders.title' })}
      content={
        <Typography.Text type="secondary">
          <FormattedMessage id="pages.planstar.orders.description" />
        </Typography.Text>
      }
    >
      <ProTable<Order, OrderQuery>
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
          const res = await services.console.ListOrders({
            keyword,
            status: optionalIntFilter(status),
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

export default OrderTable;
