// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_button.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurButton, StoryArgs<BalaurButton>>;
typedef _Scenario = BalaurButtonScenario;
typedef _Defaults = BalaurButtonDefaults;
typedef _Story = BalaurButtonStory;
typedef _Args = BalaurButtonArgs;
final BalaurButtonComponent = Component<BalaurButton, StoryArgs<BalaurButton>>(
  name: component.name ?? 'BalaurButton',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment:
      r'''Displays a beveled button that sinks when the owner presses it.''',
  stories: [
    $Primary..$generatedName = 'Primary',
    $Ghost..$generatedName = 'Ghost',
    $Wood..$generatedName = 'Wood',
    $Disabled..$generatedName = 'Disabled',
  ],
);
typedef BalaurButtonScenario = Scenario<BalaurButton, BalaurButtonArgs>;
typedef BalaurButtonDefaults = Defaults<BalaurButton, BalaurButtonArgs>;

class BalaurButtonStory extends Story<BalaurButton, BalaurButtonArgs> {
  BalaurButtonStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurButtonArgs? args,
    StoryWidgetBuilder<BalaurButton, BalaurButtonArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurButtonArgs(),
         builder:
             builder ??
             (context, args) => BalaurButton(
               key: args.key,
               label: args.label,
               onPressed: args.onPressed,
               variant: args.variant,
               size: args.size,
               leading: args.leading,
             ),
       );
}

class BalaurButtonArgs extends StoryArgs<BalaurButton> {
  BalaurButtonArgs({
    Arg<Key?>? key,
    Arg<String>? label,
    Arg<void Function()?>? onPressed,
    Arg<BalaurButtonVariant>? variant,
    Arg<BalaurButtonSize>? size,
    Arg<Widget?>? leading,
  }) : this.keyArg = $initArg('key', key, null),
       this.labelArg = $initArg('label', label, StringArg(''))!,
       this.onPressedArg = $initArg('onPressed', onPressed, null),
       this.variantArg = $initArg(
         'variant',
         variant,
         EnumArg<BalaurButtonVariant>(
           BalaurButtonVariant.primary,
           values: BalaurButtonVariant.values,
         ),
       )!,
       this.sizeArg = $initArg(
         'size',
         size,
         EnumArg<BalaurButtonSize>(
           BalaurButtonSize.regular,
           values: BalaurButtonSize.values,
         ),
       )!,
       this.leadingArg = $initArg('leading', leading, null);

  BalaurButtonArgs.fixed({
    Key? key,
    String label = '',
    void Function()? onPressed,
    BalaurButtonVariant variant = BalaurButtonVariant.primary,
    BalaurButtonSize size = BalaurButtonSize.regular,
    Widget? leading,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.labelArg = $initArg('label', Arg.fixed(label), null)!,
       this.onPressedArg = $initArg(
         'onPressed',
         onPressed == null ? null : Arg.fixed(onPressed),
         null,
       ),
       this.variantArg = $initArg('variant', Arg.fixed(variant), null)!,
       this.sizeArg = $initArg('size', Arg.fixed(size), null)!,
       this.leadingArg = $initArg(
         'leading',
         leading == null ? null : Arg.fixed(leading),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<String> labelArg;

  final Arg<void Function()?>? onPressedArg;

  final Arg<BalaurButtonVariant> variantArg;

  final Arg<BalaurButtonSize> sizeArg;

  final Arg<Widget?>? leadingArg;

  Key? get key => keyArg?.value;

  String get label => labelArg.value;

  void Function()? get onPressed => onPressedArg?.value;

  BalaurButtonVariant get variant => variantArg.value;

  BalaurButtonSize get size => sizeArg.value;

  Widget? get leading => leadingArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    labelArg,
    onPressedArg,
    variantArg,
    sizeArg,
    leadingArg,
  ];
}
