import { Form, Input, Modal } from 'antd'

export interface PasswordValues {
  old_password: string
  new_password: string
}

interface PasswordModalProps {
  open: boolean
  loading: boolean
  onCancel: () => void
  onSubmit: (values: PasswordValues) => void | Promise<void>
}

export default function PasswordModal({ open, loading, onCancel, onSubmit }: PasswordModalProps) {
  const [form] = Form.useForm()

  const handleSubmit = async () => {
    const values = await form.validateFields()
    await onSubmit({
      old_password: values.old_password as string,
      new_password: values.new_password as string,
    })
  }

  return (
    <Modal
      title="修改密码"
      open={open}
      onOk={handleSubmit}
      onCancel={onCancel}
      afterClose={() => form.resetFields()}
      confirmLoading={loading}
      destroyOnHidden
    >
      <Form form={form} layout="vertical">
        <Form.Item name="old_password" label="当前密码" rules={[{ required: true, message: '请输入当前密码' }]}>
          <Input.Password />
        </Form.Item>
        <Form.Item
          name="new_password"
          label="新密码"
          rules={[{ required: true, message: '请输入新密码' }, { min: 6, message: '至少 6 位' }]}
        >
          <Input.Password />
        </Form.Item>
        <Form.Item
          name="confirm_password"
          label="确认新密码"
          dependencies={['new_password']}
          rules={[
            { required: true, message: '请确认新密码' },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue('new_password') === value) return Promise.resolve()
                return Promise.reject(new Error('两次密码不一致'))
              },
            }),
          ]}
        >
          <Input.Password />
        </Form.Item>
      </Form>
    </Modal>
  )
}
