// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_text_field.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurTextField, StoryArgs<BalaurTextField>>;
typedef _Scenario = BalaurTextFieldScenario;
typedef _Defaults = BalaurTextFieldDefaults;
typedef _Story = BalaurTextFieldStory;
typedef _Args = BalaurTextFieldArgs;
final BalaurTextFieldComponent =
    Component<BalaurTextField, StoryArgs<BalaurTextField>>(
      name: component.name ?? 'BalaurTextField',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays a labelled parchment text field.''',
      stories: [
        $WithHint..$generatedName = 'WithHint',
        $Error..$generatedName = 'Error',
        $Disabled..$generatedName = 'Disabled',
      ],
    );
typedef BalaurTextFieldScenario =
    Scenario<BalaurTextField, BalaurTextFieldArgs>;
typedef BalaurTextFieldDefaults =
    Defaults<BalaurTextField, BalaurTextFieldArgs>;

class BalaurTextFieldStory extends Story<BalaurTextField, BalaurTextFieldArgs> {
  BalaurTextFieldStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurTextFieldArgs? args,
    StoryWidgetBuilder<BalaurTextField, BalaurTextFieldArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurTextFieldArgs(),
         builder:
             builder ??
             (context, args) => BalaurTextField(
               key: args.key,
               controller: args.controller,
               label: args.label,
               placeholder: args.placeholder,
               hint: args.hint,
               error: args.error,
               enabled: args.enabled,
               maxLines: args.maxLines,
               onChanged: args.onChanged,
               onSubmitted: args.onSubmitted,
             ),
       );
}

class BalaurTextFieldArgs extends StoryArgs<BalaurTextField> {
  BalaurTextFieldArgs({
    Arg<Key?>? key,
    Arg<TextEditingController?>? controller,
    Arg<String?>? label,
    Arg<String?>? placeholder,
    Arg<String?>? hint,
    Arg<String?>? error,
    Arg<bool>? enabled,
    Arg<int?>? maxLines,
    Arg<void Function(String)?>? onChanged,
    Arg<void Function(String)?>? onSubmitted,
  }) : this.keyArg = $initArg('key', key, null),
       this.controllerArg = $initArg('controller', controller, null),
       this.labelArg = $initArg('label', label, NullableStringArg(null))!,
       this.placeholderArg = $initArg(
         'placeholder',
         placeholder,
         NullableStringArg(null),
       )!,
       this.hintArg = $initArg('hint', hint, NullableStringArg(null))!,
       this.errorArg = $initArg('error', error, NullableStringArg(null))!,
       this.enabledArg = $initArg('enabled', enabled, BoolArg(true))!,
       this.maxLinesArg = $initArg('maxLines', maxLines, NullableIntArg(1))!,
       this.onChangedArg = $initArg('onChanged', onChanged, null),
       this.onSubmittedArg = $initArg('onSubmitted', onSubmitted, null);

  BalaurTextFieldArgs.fixed({
    Key? key,
    TextEditingController? controller,
    String? label = null,
    String? placeholder = null,
    String? hint = null,
    String? error = null,
    bool enabled = true,
    int? maxLines = 1,
    void Function(String)? onChanged,
    void Function(String)? onSubmitted,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.controllerArg = $initArg(
         'controller',
         controller == null ? null : Arg.fixed(controller),
         null,
       ),
       this.labelArg = $initArg(
         'label',
         label == null ? null : Arg.fixed(label),
         null,
       ),
       this.placeholderArg = $initArg(
         'placeholder',
         placeholder == null ? null : Arg.fixed(placeholder),
         null,
       ),
       this.hintArg = $initArg(
         'hint',
         hint == null ? null : Arg.fixed(hint),
         null,
       ),
       this.errorArg = $initArg(
         'error',
         error == null ? null : Arg.fixed(error),
         null,
       ),
       this.enabledArg = $initArg('enabled', Arg.fixed(enabled), null)!,
       this.maxLinesArg = $initArg(
         'maxLines',
         maxLines == null ? null : Arg.fixed(maxLines),
         null,
       ),
       this.onChangedArg = $initArg(
         'onChanged',
         onChanged == null ? null : Arg.fixed(onChanged),
         null,
       ),
       this.onSubmittedArg = $initArg(
         'onSubmitted',
         onSubmitted == null ? null : Arg.fixed(onSubmitted),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<TextEditingController?>? controllerArg;

  final Arg<String?>? labelArg;

  final Arg<String?>? placeholderArg;

  final Arg<String?>? hintArg;

  final Arg<String?>? errorArg;

  final Arg<bool> enabledArg;

  final Arg<int?>? maxLinesArg;

  final Arg<void Function(String)?>? onChangedArg;

  final Arg<void Function(String)?>? onSubmittedArg;

  Key? get key => keyArg?.value;

  TextEditingController? get controller => controllerArg?.value;

  String? get label => labelArg?.value;

  String? get placeholder => placeholderArg?.value;

  String? get hint => hintArg?.value;

  String? get error => errorArg?.value;

  bool get enabled => enabledArg.value;

  int? get maxLines => maxLinesArg?.value;

  void Function(String)? get onChanged => onChangedArg?.value;

  void Function(String)? get onSubmitted => onSubmittedArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    controllerArg,
    labelArg,
    placeholderArg,
    hintArg,
    errorArg,
    enabledArg,
    maxLinesArg,
    onChangedArg,
    onSubmittedArg,
  ];
}
